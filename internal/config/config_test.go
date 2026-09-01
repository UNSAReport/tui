package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigDefaults(t *testing.T) {
	tmp := t.TempDir()
	cfg, ok, err := ReadConfig(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("should not be ok for missing file")
	}
	if cfg.Prepare.Input.SrcDir != "" {
		t.Fatalf("srcDir should be empty without file, got %q", cfg.Prepare.Input.SrcDir)
	}
	// WithDefaults should fill
	cfgWith := cfg.WithDefaults()
	if cfgWith.Prepare.Input.SrcDir != DefaultSrcDir {
		t.Fatalf("srcDir %q", cfgWith.Prepare.Input.SrcDir)
	}
	if cfgWith.Prepare.Input.ReportFile != DefaultReportFile {
		t.Fatalf("reportFile %q", cfgWith.Prepare.Input.ReportFile)
	}
	if cfgWith.Capture.Columns != DefaultColumns {
		t.Fatalf("columns %d", cfgWith.Capture.Columns)
	}
	if cfgWith.Capture.Prompt != DefaultPrompt {
		t.Fatalf("prompt %q", cfgWith.Capture.Prompt)
	}
	os.WriteFile(filepath.Join(tmp, ConfigFileName), []byte(`{"mode":"invalid"}`), PermFilePublic)
	if _, _, err := ReadConfig(tmp); err == nil {
		t.Fatal("should error invalid mode")
	}
	os.WriteFile(filepath.Join(tmp, ConfigFileName), []byte(`{"mode":"multi"}`), PermFilePublic)
	if _, _, err := ReadConfig(tmp); err == nil {
		t.Fatal("should error multi without sessions")
	}
	os.WriteFile(filepath.Join(tmp, ConfigFileName), []byte(`{"template":"lab","mode":"single"}`), PermFilePublic)
	cfg, ok, err = ReadConfig(tmp)
	if err != nil || !ok {
		t.Fatalf("valid single err %v ok %v", err, ok)
	}
	if cfg.Template != "lab" {
		t.Fatalf("template %q", cfg.Template)
	}
}
