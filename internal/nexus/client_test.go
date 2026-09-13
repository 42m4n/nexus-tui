package nexus

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := New(srv.URL, false, "admin", "secret", "")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRepositoriesAndAuth(t *testing.T) {
	var gotAuth string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/service/rest/v1/repositories" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`[{"name":"maven-central","format":"maven2","type":"proxy","size":42}]`))
	}))
	repos, err := c.Repositories()
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "maven-central" || repos[0].Size != 42 {
		t.Fatalf("bad repos: %+v", repos)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	if gotAuth != want {
		t.Fatalf("auth = %q, want %q", gotAuth, want)
	}
}

func TestPagination(t *testing.T) {
	var calls int
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		tok := r.URL.Query().Get("continuationToken")
		if tok == "" {
			w.Write([]byte(`{"items":[{"id":"a"}],"continuationToken":"next"}`))
			return
		}
		if tok != "next" {
			t.Errorf("token = %q", tok)
		}
		w.Write([]byte(`{"items":[{"id":"b"}],"continuationToken":null}`))
	}))
	comps, err := c.Components("maven-central")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(comps) != 2 || comps[0].ID != "a" || comps[1].ID != "b" {
		t.Fatalf("calls=%d comps=%+v", calls, comps)
	}
}

func TestStatusReadOnly(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/service/rest/v1/status/writable" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	s, err := c.Status()
	if err != nil {
		t.Fatal(err)
	}
	if s != "read-only" {
		t.Fatalf("status = %q", s)
	}
}

func TestDeleteGuarded(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called without --allow-writes")
	}))
	if err := c.Delete("/repositories/x"); err == nil {
		t.Fatal("expected write guard error")
	}
	c.Writes = true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c2, _ := New(srv.URL, false, "", "", "")
	c2.Writes = true
	if err := c2.Delete("/repositories/x"); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidateCache(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server must not be called without --allow-writes")
	}))
	if err := c.InvalidateCache("maven-central"); err == nil {
		t.Fatal("expected write guard error")
	}
	c.Writes = true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/service/rest/v1/repositories/maven-central/invalidate-cache" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c2, _ := New(srv.URL, false, "", "", "")
	c2.Writes = true
	if err := c2.InvalidateCache("maven-central"); err != nil {
		t.Fatal(err)
	}
}

func TestSearchPassesFilters(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("name") != "jackson" || q.Get("repository") != "maven-central" {
			t.Errorf("query = %v", q)
		}
		w.Write([]byte(`{"items":[{"id":"c1","name":"jackson"}],"continuationToken":null}`))
	}))
	q := url.Values{"name": {"jackson"}, "repository": {"maven-central"}}
	comps, err := c.Search(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(comps) != 1 || comps[0].Name != "jackson" {
		t.Fatalf("comps = %+v", comps)
	}
}

func TestRepoStatus(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/service/rest/v1/repositories/maven-central/status" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"healthy":false,"description":"Remote unavailable: connection timed out"}`))
	}))
	st, err := c.RepoStatus("maven-central")
	if err != nil {
		t.Fatal(err)
	}
	if st.Healthy || st.Description != "Remote unavailable: connection timed out" {
		t.Fatalf("status = %+v", st)
	}
	if _, err := c.RepoStatus(""); err == nil {
		t.Fatal("expected error for empty repository")
	}
}

func TestReadOnly(t *testing.T) {
	t.Run("read-only mode off", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/service/rest/v1/read-only" {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.Write([]byte(`{"readOnly":false}`))
		}))
		ro, err := c.ReadOnly()
		if err != nil {
			t.Fatal(err)
		}
		if ro {
			t.Fatal("readOnly = true, want false")
		}
	})
	t.Run("read-only mode on", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"readOnly":true}`))
		}))
		ro, err := c.ReadOnly()
		if err != nil {
			t.Fatal(err)
		}
		if !ro {
			t.Fatal("readOnly = false, want true")
		}
	})
}

func TestErrorIncludesStatus(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	_, err := c.Repositories()
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
}
