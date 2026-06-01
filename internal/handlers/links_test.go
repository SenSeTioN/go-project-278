package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	"github.com/SenSeTioN/go-project-278/internal/db"
)

type mockQuerier struct {
	listFn            func(ctx context.Context) ([]db.Link, error)
	listRangeFn       func(ctx context.Context, arg db.ListLinksRangeParams) ([]db.Link, error)
	countFn           func(ctx context.Context) (int64, error)
	getFn             func(ctx context.Context, id int64) (db.Link, error)
	getByNameFn       func(ctx context.Context, name string) (db.Link, error)
	createFn          func(ctx context.Context, arg db.CreateLinkParams) (db.Link, error)
	updateFn          func(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error)
	deleteFn          func(ctx context.Context, id int64) (int64, error)
	createVisitFn     func(ctx context.Context, arg db.CreateLinkVisitParams) (db.LinkVisit, error)
	listVisitsFn      func(ctx context.Context) ([]db.LinkVisit, error)
	listVisitsRangeFn func(ctx context.Context, arg db.ListLinkVisitsRangeParams) ([]db.LinkVisit, error)
	countVisitsFn     func(ctx context.Context) (int64, error)
}

func (m *mockQuerier) ListLinks(ctx context.Context) ([]db.Link, error) {
	return m.listFn(ctx)
}
func (m *mockQuerier) ListLinksRange(ctx context.Context, arg db.ListLinksRangeParams) ([]db.Link, error) {
	return m.listRangeFn(ctx, arg)
}
func (m *mockQuerier) CountLinks(ctx context.Context) (int64, error) {
	return m.countFn(ctx)
}
func (m *mockQuerier) GetLink(ctx context.Context, id int64) (db.Link, error) {
	return m.getFn(ctx, id)
}
func (m *mockQuerier) GetLinkByShortName(ctx context.Context, name string) (db.Link, error) {
	return m.getByNameFn(ctx, name)
}
func (m *mockQuerier) CreateLink(ctx context.Context, arg db.CreateLinkParams) (db.Link, error) {
	return m.createFn(ctx, arg)
}
func (m *mockQuerier) UpdateLink(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error) {
	return m.updateFn(ctx, arg)
}
func (m *mockQuerier) DeleteLink(ctx context.Context, id int64) (int64, error) {
	return m.deleteFn(ctx, id)
}
func (m *mockQuerier) CreateLinkVisit(ctx context.Context, arg db.CreateLinkVisitParams) (db.LinkVisit, error) {
	return m.createVisitFn(ctx, arg)
}
func (m *mockQuerier) ListLinkVisits(ctx context.Context) ([]db.LinkVisit, error) {
	return m.listVisitsFn(ctx)
}
func (m *mockQuerier) ListLinkVisitsRange(ctx context.Context, arg db.ListLinkVisitsRangeParams) ([]db.LinkVisit, error) {
	return m.listVisitsRangeFn(ctx, arg)
}
func (m *mockQuerier) CountLinkVisits(ctx context.Context) (int64, error) {
	return m.countVisitsFn(ctx)
}

func newRouter(q db.Querier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(q, "https://short.io").Register(r)
	return r
}

func do(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestList(t *testing.T) {
	mock := &mockQuerier{
		countFn: func(ctx context.Context) (int64, error) { return 2, nil },
		listFn: func(ctx context.Context) ([]db.Link, error) {
			return []db.Link{
				{ID: 1, OriginalUrl: "https://a.com", ShortName: "aaa"},
				{ID: 2, OriginalUrl: "https://b.com", ShortName: "bbb"},
			}, nil
		},
	}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	if cr := w.Header().Get("Content-Range"); cr != "links 0-2/2" {
		t.Errorf("Content-Range: want %q, got %q", "links 0-2/2", cr)
	}
	var got []linkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len: want 2, got %d", len(got))
	}
	if got[0].ShortURL != "https://short.io/r/aaa" {
		t.Errorf("short_url: %q", got[0].ShortURL)
	}
}

func TestList_Range(t *testing.T) {
	all := make([]db.Link, 42)
	for i := range all {
		all[i] = db.Link{ID: int64(i + 1), OriginalUrl: "u", ShortName: "s"}
	}
	mock := &mockQuerier{
		countFn: func(ctx context.Context) (int64, error) { return int64(len(all)), nil },
		listRangeFn: func(ctx context.Context, arg db.ListLinksRangeParams) ([]db.Link, error) {
			from, to := int(arg.Offset), int(arg.Offset+arg.Limit)
			if to > len(all) {
				to = len(all)
			}
			return all[from:to], nil
		},
	}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links?range=[0,10]", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if cr := w.Header().Get("Content-Range"); cr != "links 0-10/42" {
		t.Errorf("Content-Range: want %q, got %q", "links 0-10/42", cr)
	}
	var got []linkResponse
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 10 {
		t.Fatalf("len: want 10, got %d", len(got))
	}
	if got[0].ID != 1 || got[9].ID != 10 {
		t.Errorf("ids: want 1..10, got %d..%d", got[0].ID, got[9].ID)
	}
}

func TestList_RangeOffset(t *testing.T) {
	all := make([]db.Link, 11)
	for i := range all {
		all[i] = db.Link{ID: int64(i + 1), OriginalUrl: "u", ShortName: "s"}
	}
	mock := &mockQuerier{
		countFn: func(ctx context.Context) (int64, error) { return int64(len(all)), nil },
		listRangeFn: func(ctx context.Context, arg db.ListLinksRangeParams) ([]db.Link, error) {
			from, to := int(arg.Offset), int(arg.Offset+arg.Limit)
			if to > len(all) {
				to = len(all)
			}
			return all[from:to], nil
		},
	}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links?range=[5,10]", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if cr := w.Header().Get("Content-Range"); cr != "links 5-10/11" {
		t.Errorf("Content-Range: want %q, got %q", "links 5-10/11", cr)
	}
	var got []linkResponse
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 5 {
		t.Fatalf("len: want 5, got %d", len(got))
	}
	if got[0].ID != 6 || got[4].ID != 10 {
		t.Errorf("ids: want 6..10, got %d..%d", got[0].ID, got[4].ID)
	}
}

func TestList_InvalidRange(t *testing.T) {
	mock := &mockQuerier{
		countFn: func(ctx context.Context) (int64, error) { return 0, nil },
	}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links?range=oops", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", w.Code)
	}
}

func TestCreate_WithShortName(t *testing.T) {
	mock := &mockQuerier{
		createFn: func(ctx context.Context, arg db.CreateLinkParams) (db.Link, error) {
			if arg.ShortName != "custom" {
				t.Errorf("short_name: want custom, got %q", arg.ShortName)
			}
			return db.Link{ID: 10, OriginalUrl: arg.OriginalUrl, ShortName: arg.ShortName}, nil
		},
	}
	body := map[string]string{"original_url": "https://x.com", "short_name": "custom"}
	w := do(t, newRouter(mock), http.MethodPost, "/api/links", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCreate_AutoShortName(t *testing.T) {
	mock := &mockQuerier{
		createFn: func(ctx context.Context, arg db.CreateLinkParams) (db.Link, error) {
			if len(arg.ShortName) == 0 {
				t.Error("short_name should be generated")
			}
			return db.Link{ID: 11, OriginalUrl: arg.OriginalUrl, ShortName: arg.ShortName}, nil
		},
	}
	h := New(mock, "https://short.io")
	h.NameGen = func(int) (string, error) { return "GENERATED", nil }

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.Register(r)

	w := do(t, r, http.MethodPost, "/api/links", map[string]string{"original_url": "https://x.com"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status: %d", w.Code)
	}
	var got linkResponse
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.ShortName != "GENERATED" {
		t.Errorf("short_name: %q", got.ShortName)
	}
}

func TestCreate_MissingFields(t *testing.T) {
	mock := &mockQuerier{}
	w := do(t, newRouter(mock), http.MethodPost, "/api/links", map[string]string{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: want 422, got %d", w.Code)
	}
	var got struct {
		Errors map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Errors["original_url"]; !ok {
		t.Errorf("expected errors.original_url, got %#v", got.Errors)
	}
}

func TestCreate_InvalidURL(t *testing.T) {
	mock := &mockQuerier{}
	body := map[string]string{"original_url": "not-a-url"}
	w := do(t, newRouter(mock), http.MethodPost, "/api/links", body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: %d", w.Code)
	}
	var got struct {
		Errors map[string]string `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	msg, ok := got.Errors["original_url"]
	if !ok {
		t.Fatalf("missing errors.original_url: %#v", got.Errors)
	}
	if !strings.Contains(msg, "'url' tag") && !strings.Contains(msg, "url") {
		t.Errorf("unexpected message: %q", msg)
	}
}

func TestCreate_ShortNameTooShort(t *testing.T) {
	mock := &mockQuerier{}
	body := map[string]string{"original_url": "https://x.com", "short_name": "ab"}
	w := do(t, newRouter(mock), http.MethodPost, "/api/links", body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: %d", w.Code)
	}
	var got struct {
		Errors map[string]string `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if _, ok := got.Errors["short_name"]; !ok {
		t.Errorf("expected errors.short_name, got %#v", got.Errors)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	mock := &mockQuerier{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(mock, "https://short.io").Register(r)

	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid request") {
		t.Errorf("body: %s", w.Body.String())
	}
}

func TestCreate_DuplicateShortName(t *testing.T) {
	mock := &mockQuerier{
		createFn: func(ctx context.Context, arg db.CreateLinkParams) (db.Link, error) {
			return db.Link{}, &pq.Error{Code: "23505"}
		},
	}
	body := map[string]string{"original_url": "https://x.com", "short_name": "taken"}
	w := do(t, newRouter(mock), http.MethodPost, "/api/links", body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: %d", w.Code)
	}
	var got struct {
		Errors map[string]string `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Errors["short_name"] != "short name already in use" {
		t.Errorf("errors.short_name: %q", got.Errors["short_name"])
	}
}

func TestGet_OK(t *testing.T) {
	mock := &mockQuerier{
		getFn: func(ctx context.Context, id int64) (db.Link, error) {
			return db.Link{ID: id, OriginalUrl: "https://a.com", ShortName: "aaa"}, nil
		},
	}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	mock := &mockQuerier{
		getFn: func(ctx context.Context, id int64) (db.Link, error) {
			return db.Link{}, sql.ErrNoRows
		},
	}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links/999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestGet_InvalidID(t *testing.T) {
	mock := &mockQuerier{}
	w := do(t, newRouter(mock), http.MethodGet, "/api/links/not-a-number", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestUpdate_OK(t *testing.T) {
	mock := &mockQuerier{
		updateFn: func(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error) {
			return db.Link{ID: arg.ID, OriginalUrl: arg.OriginalUrl, ShortName: arg.ShortName}, nil
		},
	}
	body := map[string]string{"original_url": "https://new.com", "short_name": "new"}
	w := do(t, newRouter(mock), http.MethodPut, "/api/links/1", body)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	mock := &mockQuerier{
		updateFn: func(ctx context.Context, arg db.UpdateLinkParams) (db.Link, error) {
			return db.Link{}, sql.ErrNoRows
		},
	}
	body := map[string]string{"original_url": "https://new.com", "short_name": "new"}
	w := do(t, newRouter(mock), http.MethodPut, "/api/links/999", body)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestDelete_OK(t *testing.T) {
	mock := &mockQuerier{
		deleteFn: func(ctx context.Context, id int64) (int64, error) {
			return 1, nil
		},
	}
	w := do(t, newRouter(mock), http.MethodDelete, "/api/links/1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status: %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	mock := &mockQuerier{
		deleteFn: func(ctx context.Context, id int64) (int64, error) {
			return 0, nil
		},
	}
	w := do(t, newRouter(mock), http.MethodDelete, "/api/links/999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: %d", w.Code)
	}
}
