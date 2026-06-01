package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/SenSeTioN/go-project-278/internal/db"
)

// Visits обслуживает HTTP-эндпоинт списка посещений (/api/link_visits).
// Сами записи о посещениях создаются в Links.redirect — Visits отвечает только за чтение.
type Visits struct {
	Queries db.Querier
}

// NewVisits создаёт обработчик списка посещений.
func NewVisits(q db.Querier) *Visits {
	return &Visits{Queries: q}
}

// Register регистрирует маршрут GET /api/link_visits на переданном роутере.
func (h *Visits) Register(r gin.IRouter) {
	r.GET("/api/link_visits", h.list)
}

// list возвращает посещения с пагинацией и заголовком Content-Range.
func (h *Visits) list(c *gin.Context) {
	ctx := c.Request.Context()

	total, err := h.Queries.CountLinkVisits(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rawRange := c.Query("range")
	if rawRange == "" {
		rawRange = c.GetHeader("Range")
	}

	var visits []db.LinkVisit
	var from, to int64

	if rawRange == "" {
		visits, err = h.Queries.ListLinkVisits(ctx)
		from, to = 0, total
	} else {
		var ok bool
		from, to, ok = parseRange(rawRange)
		if !ok || to < from {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range, expected [from,to]"})
			return
		}
		visits, err = h.Queries.ListLinkVisitsRange(ctx, db.ListLinkVisitsRangeParams{
			Limit:  int32(to - from),
			Offset: int32(from),
		})
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Range", fmt.Sprintf("link_visits %d-%d/%d", from, to, total))
	c.Header("Accept-Ranges", "link_visits")

	if visits == nil {
		visits = []db.LinkVisit{}
	}
	c.JSON(http.StatusOK, visits)
}
