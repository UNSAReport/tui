package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigDefaults(t *testing.T) {
	tmp := t.TempDir()
	// empty dir -> no config, should apply defaults
	cfg, ok, err := ReadConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("should not be ok for missing file")
	}
	if cfg.Prepare.Input.SrcDir != "src" {
		t.Fatalf("srcDir %q", cfg.Prepare.Input.SrcDir)
	}
	if cfg.Prepare.Input.ReportFile != "report.typ" {
		t.Fatalf("reportFile %q", cfg.Prepare.Input.ReportFile)
	}
	if cfg.Capture.Columns != 120 {
		t.Fatalf("columns %d", cfg.Capture.Columns)
	}
	if cfg.Capture.Prompt != "❯ " {
		t.Fatalf("prompt %q", cfg.Capture.Prompt)
	}
	// invalid mode
	os.WriteFile(filepath.Join(tmp, "unsareport.json"), []byte(`{"mode":"invalid"}`), 0o644)
	if _, _, err := ReadConfig(tmp); err == nil {
		t.Fatal("should error invalid mode")
	}
	// multi without sessions
	os.WriteFile(filepath.Join(tmp, "unsareport.json"), []byte(`{"mode":"multi"}`), 0o644)
	if _, _, err := ReadConfig(tmp); err == nil {
		t.Fatal("should error multi without sessions")
	}
	// valid single
	os.WriteFile(filepath.Join(tmp, "unsareport.json"), []byte(`{"template":"lab","mode":"single"}`), 0o644)
	cfg, ok, err = ReadConfig(tmp)
	if err != nil || !ok {
		t.Fatalf("valid single err %v ok %v", err, ok)
	}
	if cfg.Template != "lab" {
		t.Fatalf("template %q", cfg.Template)
	}
}
