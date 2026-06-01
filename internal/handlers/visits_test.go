package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/SenSeTioN/go-project-278/internal/db"
)

func newRouterWithVisits(q db.Querier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	New(q, "https://short.io").Register(r)
	NewVisits(q).Register(r)
	return r
}

func TestRedirect_OK(t *testing.T) {
	var recorded db.CreateLinkVisitParams
	mock := &mockQuerier{
		getByNameFn: func(ctx context.Context, name string) (db.Link, error) {
			if name != "abc" {
				t.Errorf("short_name: want abc, got %q", name)
			}
			return db.Link{ID: 7, OriginalUrl: "https://example.com/long", ShortName: "abc"}, nil
		},
		createVisitFn: func(ctx context.Context, arg db.CreateLinkVisitParams) (db.LinkVisit, error) {
			recorded = arg
			return db.LinkVisit{ID: 1, LinkID: arg.LinkID, Status: arg.Status}, nil
		},
	}

	w := do(t, newRouterWithVisits(mock), http.MethodGet, "/r/abc", nil)

	if w.Code != http.StatusFound {
		t.Fatalf("status: want 302, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/long" {
		t.Errorf("Location: %q", loc)
	}
	if recorded.LinkID != 7 {
		t.Errorf("visit.link_id: want 7, got %d", recorded.LinkID)
	}
	if recorded.Status != http.StatusFound {
		t.Errorf("visit.status: want 302, got %d", recorded.Status)
	}
}

func TestRedirect_NotFound(t *testing.T) {
	mock := &mockQuerier{
		getByNameFn: func(ctx context.Context, name string) (db.Link, error) {
			return db.Link{}, sql.ErrNoRows
		},
	}
	w := do(t, newRouterWithVisits(mock), http.MethodGet, "/r/unknown", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", w.Code)
	}
}

func TestVisits_List(t *testing.T) {
	mock := &mockQuerier{
		countVisitsFn: func(ctx context.Context) (int64, error) { return 2, nil },
		listVisitsFn: func(ctx context.Context) ([]db.LinkVisit, error) {
			return []db.LinkVisit{
				{ID: 1, LinkID: 1, Ip: "1.1.1.1", UserAgent: "curl", Status: 302},
				{ID: 2, LinkID: 1, Ip: "2.2.2.2", UserAgent: "wget", Status: 302},
			}, nil
		},
	}
	w := do(t, newRouterWithVisits(mock), http.MethodGet, "/api/link_visits", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if cr := w.Header().Get("Content-Range"); cr != "link_visits 0-2/2" {
		t.Errorf("Content-Range: %q", cr)
	}
	var got []db.LinkVisit
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 2 {
		t.Fatalf("len: %d", len(got))
	}
}

func TestVisits_RangeQuery(t *testing.T) {
	all := make([]db.LinkVisit, 357)
	for i := range all {
		all[i] = db.LinkVisit{ID: int64(i + 1), LinkID: 1, Status: 302}
	}
	mock := &mockQuerier{
		countVisitsFn: func(ctx context.Context) (int64, error) { return int64(len(all)), nil },
		listVisitsRangeFn: func(ctx context.Context, arg db.ListLinkVisitsRangeParams) ([]db.LinkVisit, error) {
			from, to := int(arg.Offset), int(arg.Offset+arg.Limit)
			if to > len(all) {
				to = len(all)
			}
			return all[from:to], nil
		},
	}
	w := do(t, newRouterWithVisits(mock), http.MethodGet, "/api/link_visits?range=[10,20]", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if cr := w.Header().Get("Content-Range"); cr != "link_visits 10-20/357" {
		t.Errorf("Content-Range: %q", cr)
	}
	var got []db.LinkVisit
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 10 {
		t.Fatalf("len: want 10, got %d", len(got))
	}
}

func TestVisits_RangeHeader(t *testing.T) {
	all := make([]db.LinkVisit, 100)
	for i := range all {
		all[i] = db.LinkVisit{ID: int64(i + 1), LinkID: 1, Status: 302}
	}
	mock := &mockQuerier{
		countVisitsFn: func(ctx context.Context) (int64, error) { return int64(len(all)), nil },
		listVisitsRangeFn: func(ctx context.Context, arg db.ListLinkVisitsRangeParams) ([]db.LinkVisit, error) {
			from, to := int(arg.Offset), int(arg.Offset+arg.Limit)
			if to > len(all) {
				to = len(all)
			}
			return all[from:to], nil
		},
	}
	r := newRouterWithVisits(mock)
	req, _ := http.NewRequest(http.MethodGet, "/api/link_visits", nil)
	req.Header.Set("Range", "[0,5]")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}
	if cr := w.Header().Get("Content-Range"); cr != "link_visits 0-5/100" {
		t.Errorf("Content-Range: %q", cr)
	}
}
