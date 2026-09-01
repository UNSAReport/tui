package install

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/UNSAReport/tui/internal/registry"
)

func TestInstallLab(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/packages/lab":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"name": "lab", "description": "lab", "version": "1.0.0",
				"versions": []string{"1.0.0"},
			})
			return
		case "/v1/packages/lab/versions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"versions": []map[string]any{{"version": "1.0.0"}},
			})
			return
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	t.Setenv("UNSAREP_REGISTRY_URL", srv.URL)
	// Ensure registry client uses mock URL
	_ = registry.NewClient
	tmp := t.TempDir()
	dest := tmp + "/newproj"
	// Create minimal local template dir for fetchFiles
	localDir := tmp + "/tmpl"
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(localDir+"/manifest.json", []byte(`{"mode":"single","entries":[{"kind":"file","src":"report.typ","dest":"report.typ"},{"kind":"file","src":"lib.typ","dest":"lib.typ"}]}`), 0o644)
	_ = os.WriteFile(localDir+"/report.typ", []byte("report"), 0o644)
	_ = os.WriteFile(localDir+"/lib.typ", []byte("lib"), 0o644)
	if err := Execute(context.Background(), Options{TemplateArg: "lab@1.0.0", Dest: dest, Local: localDir}); err != nil {
		t.Fatalf("install err: %v", err)
	}
	for _, f := range []string{"report.typ", "unsareport.json", "lib.typ"} {
		if _, err := os.Stat(dest + "/" + f); err != nil {
			t.Fatalf("missing %s: %v", f, err)
		}
	}
	if err := Execute(context.Background(), Options{TemplateArg: "lab@not-semver", Dest: tmp + "/newproj2"}); err == nil {
		t.Fatal("should have failed invalid semver")
	}
}
func TestInvalidSemver(t *testing.T) {
	tmp := t.TempDir()
	err := Execute(context.Background(), Options{TemplateArg: "lab@not-semver", Dest: tmp + "/x"})
	if err == nil || !strings.Contains(err.Error(), "invalid version constraint") {
		t.Fatalf("expected invalid version constraint error, got %v", err)
	}
}
