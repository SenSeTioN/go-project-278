// Package handlers содержит HTTP-обработчики API: ссылки (/api/links),
// редирект по короткому имени (/r/:code) и список посещений
package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/lib/pq"

	"github.com/SenSeTioN/go-project-278/internal/db"
	"github.com/SenSeTioN/go-project-278/internal/shortener"
)

// init настраивает движок валидации Gin: имена полей в ошибках валидации
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

// uniqueViolation — SQLSTATE-код PostgreSQL для нарушения уникальности
const uniqueViolation = "23505"

// Links — обработчик CRUD-эндпоинтов ссылок и редиректа /r/:code.
type Links struct {
	Queries db.Querier
	BaseURL string
	NameGen func(int) (string, error)
}

// New создаёт обработчик ссылок. Хвостовой слэш в baseURL обрезается,
func New(q db.Querier, baseURL string) *Links {
	return &Links{
		Queries: q,
		BaseURL: strings.TrimRight(baseURL, "/"),
		NameGen: shortener.Generate,
	}
}

// linkResponse — формат сущности ссылки в JSON-ответе. Поле ShortURL
type linkResponse struct {
	ID          int64  `json:"id"`
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
	ShortURL    string `json:"short_url"`
}

// createLinkPayload — общий формат тела запроса для POST и PUT.
//
// Правила валидации:
//   - original_url: обязательное, должно быть валидным URL по RFC 3986;
//   - short_name: опциональное; если задано — длина от 3 до 32 символов.
//     Если не задано, сервер сгенерирует значение через NameGen.
type createLinkPayload struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
	ShortName   string `json:"short_name"   binding:"omitempty,min=3,max=32"`
}

// bindPayload разбирает JSON-тело запроса в dst и валидирует его.
// При ошибке функция сама пишет ответ клиенту и возвращает false:
//   - 422 + {"errors": {field: message}} — при ошибках валидации
//     (validator.ValidationErrors);
//   - 400 + {"error": "invalid request"} — при некорректном JSON
//     (синтаксис, типы, неподходящее тело).
//
// Возврат true означает, что dst заполнен и можно продолжать обработку.
func bindPayload(c *gin.Context, dst *createLinkPayload) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			out := make(map[string]string, len(verrs))
			for _, fe := range verrs {
				out[fe.Field()] = fe.Error()
			}
			c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": out})
			return false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return false
	}
	return true
}

// shortNameTakenResponse отправляет ответ 422 с сообщением о занятом
// short_name. Используется при unique_violation от БД, чтобы клиент
func shortNameTakenResponse(c *gin.Context) {
	c.JSON(http.StatusUnprocessableEntity, gin.H{
		"errors": gin.H{"short_name": "short name already in use"},
	})
}

// toResponse преобразует доменную модель db.Link в JSON-представление.
// Поле ShortURL собирается из BaseURL и ShortName в формате
// "{base}/r/{short_name}".
func (h *Links) toResponse(l db.Link) linkResponse {
	return linkResponse{
		ID:          l.ID,
		OriginalURL: l.OriginalUrl,
		ShortName:   l.ShortName,
		ShortURL:    fmt.Sprintf("%s/r/%s", h.BaseURL, l.ShortName),
	}
}

// Register регистрирует CRUD-маршруты ссылок под /api/links и
// публичный маршрут редиректа /r/:code на переданном роутере.
func (h *Links) Register(r gin.IRouter) {
	g := r.Group("/api/links")
	g.GET("", h.list)
	g.POST("", h.create)
	g.GET("/:id", h.get)
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.delete)

	r.GET("/r/:code", h.redirect)
}

// redirect обслуживает GET /r/:code: ищет ссылку по короткому имени
func (h *Links) redirect(c *gin.Context) {
	code := c.Param("code")
	ctx := c.Request.Context()

	link, err := h.Queries.GetLinkByShortName(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.String(http.StatusNotFound, "link not found")
			return
		}
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	status := http.StatusFound
	if _, err := h.Queries.CreateLinkVisit(ctx, db.CreateLinkVisitParams{
		LinkID:    link.ID,
		Ip:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Referer:   c.GetHeader("Referer"),
		Status:    int32(status),
	}); err != nil {
		log.Printf("record visit: %v", err)
	}

	c.Redirect(status, link.OriginalUrl)
}

// rangeRe — шаблон для значения параметра range: [from,to] с допустимыми пробелами вокруг запятой. Используется как query (?range=[0,10])
var rangeRe = regexp.MustCompile(`^\[\s*(\d+)\s*,\s*(\d+)\s*\]$`)

// parseRange разбирает строку формата "[from,to]" в пару целых чисел.
func parseRange(raw string) (from, to int64, ok bool) {
	m := rangeRe.FindStringSubmatch(raw)
	if m == nil {
		return 0, 0, false
	}
	from, _ = strconv.ParseInt(m[1], 10, 64)
	to, _ = strconv.ParseInt(m[2], 10, 64)
	return from, to, true
}

// list возвращает список ссылок с пагинацией.
func (h *Links) list(c *gin.Context) {
	ctx := c.Request.Context()

	total, err := h.Queries.CountLinks(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rawRange := c.Query("range")
	var links []db.Link
	var from, to int64

	if rawRange == "" {
		links, err = h.Queries.ListLinks(ctx)
		from, to = 0, total
	} else {
		var ok bool
		from, to, ok = parseRange(rawRange)
		if !ok || to < from {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range, expected [from,to]"})
			return
		}
		limit := to - from
		links, err = h.Queries.ListLinksRange(ctx, db.ListLinksRangeParams{
			Limit:  int32(limit),
			Offset: int32(from),
		})
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Range", fmt.Sprintf("links %d-%d/%d", from, to, total))
	c.Header("Accept-Ranges", "links")

	resp := make([]linkResponse, 0, len(links))
	for _, l := range links {
		resp = append(resp, h.toResponse(l))
	}
	c.JSON(http.StatusOK, resp)
}

// create обрабатывает POST /api/links.
func (h *Links) create(c *gin.Context) {
	var req createLinkPayload
	if !bindPayload(c, &req) {
		return
	}

	shortName := req.ShortName
	if shortName == "" {
		generated, err := h.NameGen(shortener.DefaultLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate short name"})
			return
		}
		shortName = generated
	}

	link, err := h.Queries.CreateLink(c.Request.Context(), db.CreateLinkParams{
		OriginalUrl: req.OriginalURL,
		ShortName:   shortName,
	})
	if err != nil {
		if isUniqueViolation(err) {
			shortNameTakenResponse(c)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, h.toResponse(link))
}

// get обрабатывает GET /api/links/:id.
func (h *Links) get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	link, err := h.Queries.GetLink(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toResponse(link))
}

// update обрабатывает PUT /api/links/:id.
func (h *Links) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req createLinkPayload
	if !bindPayload(c, &req) {
		return
	}
	shortName := req.ShortName
	if shortName == "" {
		generated, err := h.NameGen(shortener.DefaultLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate short name"})
			return
		}
		shortName = generated
	}
	link, err := h.Queries.UpdateLink(c.Request.Context(), db.UpdateLinkParams{
		ID:          id,
		OriginalUrl: req.OriginalURL,
		ShortName:   shortName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		if isUniqueViolation(err) {
			shortNameTakenResponse(c)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toResponse(link))
}

// delete обрабатывает DELETE /api/links/:id.
func (h *Links) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	rows, err := h.Queries.DeleteLink(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// parseID извлекает параметр :id из URL и парсит его как int64.
func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// isUniqueViolation возвращает true, если err — это ошибка PostgreSQL
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == uniqueViolation
	}
	return false
}
