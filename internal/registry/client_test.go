package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListTemplates_Hono(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/packages" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"packages": []map[string]any{
					{"name": "lab", "description": "lab desc"},
					{"name": "multi-lab", "description": "multi desc"},
				},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	templates, err := c.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("err %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("len %d", len(templates))
	}
	if templates[0].Name != "lab" {
		t.Fatalf("name %q", templates[0].Name)
	}
}

func TestGetTemplateVersion_Semver(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/packages/lab":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name":        "lab",
				"description": "lab",
				"versions":    []string{"1.0.0", "1.1.0", "2.0.0"},
			})
		case "/v1/packages/lab/versions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"versions": []map[string]any{{"version": "1.0.0"}, {"version": "1.1.0"}, {"version": "2.0.0"}},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	info, err := c.GetTemplateVersion(context.Background(), "lab", "^1.0.0")
	if err != nil {
		t.Fatalf("err %v", err)
	}
	if info.Version != "1.1.0" {
		t.Fatalf("resolved %q", info.Version)
	}
}
