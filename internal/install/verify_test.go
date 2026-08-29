package install

import (
	"context"
	"os"
	"testing"
)

func TestInstallLab(t *testing.T) {
	tmp := t.TempDir()
	dest := tmp + "/newproj"
	if err := Execute(context.Background(), Options{TemplateArg: "lab@1.0.0", Dest: dest}); err != nil {
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
