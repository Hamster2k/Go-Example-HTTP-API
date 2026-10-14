package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return NewServer(store)
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestCRUDLifecycle(t *testing.T) {
	h := newTestServer(t)

	rec := do(t, h, "POST", "/objects", `{"name":" first ","message":"hello"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d: %s", rec.Code, rec.Body)
	}
	created := decode[Object](t, rec)
	if created.ID != 1 || created.Name != "first" || created.Message != "hello" {
		t.Fatalf("unexpected object: %+v", created)
	}
	if loc := rec.Header().Get("Location"); loc != "/objects/1" {
		t.Errorf("Location = %q", loc)
	}

	rec = do(t, h, "GET", "/objects/1", "")
	if rec.Code != http.StatusOK || decode[Object](t, rec).Name != "first" {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}

	rec = do(t, h, "PUT", "/objects/1", `{"name":"renamed","message":"updated"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if got := decode[Object](t, rec); got.Name != "renamed" || got.Message != "updated" {
		t.Fatalf("unexpected update: %+v", got)
	}

	rec = do(t, h, "GET", "/objects", "")
	if list := decode[[]Object](t, rec); rec.Code != http.StatusOK || len(list) != 1 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}

	if rec = do(t, h, "DELETE", "/objects/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec = do(t, h, "GET", "/objects/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: %d", rec.Code)
	}
}

func TestEmptyListIsArray(t *testing.T) {
	rec := do(t, newTestServer(t), "GET", "/objects", "")
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("got %q", rec.Body)
	}
}

func TestErrors(t *testing.T) {
	h := newTestServer(t)
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"invalid json", "POST", "/objects", `{nope`, 400},
		{"unknown field", "POST", "/objects", `{"name":"a","message":"b","x":1}`, 400},
		{"trailing data", "POST", "/objects", `{"name":"a","message":"b"}{}`, 400},
		{"missing name", "POST", "/objects", `{"message":"b"}`, 422},
		{"blank message", "POST", "/objects", `{"name":"a","message":"  "}`, 422},
		{"name too long", "POST", "/objects", `{"name":"` + strings.Repeat("a", 101) + `","message":"b"}`, 422},
		{"body too large", "POST", "/objects", `{"name":"a","message":"` + strings.Repeat("a", maxBodyBytes) + `"}`, 413},
		{"bad id", "GET", "/objects/abc", "", 400},
		{"zero id", "GET", "/objects/0", "", 400},
		{"get missing", "GET", "/objects/99", "", 404},
		{"put missing", "PUT", "/objects/99", `{"name":"a","message":"b"}`, 404},
		{"delete missing", "DELETE", "/objects/99", "", 404},
		{"bad limit", "GET", "/objects?limit=0", "", 400},
		{"limit too large", "GET", "/objects?limit=1000", "", 400},
		{"bad offset", "GET", "/objects?offset=-1", "", 400},
		{"method not allowed", "PATCH", "/objects/1", "", 405},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if rec := do(t, h, tc.method, tc.path, tc.body); rec.Code != tc.want {
				t.Errorf("got %d, want %d: %s", rec.Code, tc.want, rec.Body)
			}
		})
	}
}

func TestPagination(t *testing.T) {
	h := newTestServer(t)
	for i := 0; i < 3; i++ {
		do(t, h, "POST", "/objects", `{"name":"n","message":"m"}`)
	}
	list := decode[[]Object](t, do(t, h, "GET", "/objects?limit=2&offset=1", ""))
	if len(list) != 2 || list[0].ID != 2 || list[1].ID != 3 {
		t.Fatalf("unexpected page: %+v", list)
	}
}

func TestHealth(t *testing.T) {
	if rec := do(t, newTestServer(t), "GET", "/healthz", ""); rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}
