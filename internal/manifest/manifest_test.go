package manifest

import "testing"

func TestLoadAndValidate(t *testing.T) {
	data := []byte(`{"mode":"single","entries":[{"kind":"file","src":"a","dest":"b"}]}`)
	m, err := LoadAndValidateManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.Mode != "single" {
		t.Fatalf("mode %q", m.Mode)
	}
	bad := []byte(`{"mode":"single","entries":[{"kind":"file","dest":"b"}]}`)
	if _, err := LoadAndValidateManifest(bad); err == nil {
		t.Fatal("should fail validation")
	}
	multi := []byte(`{"mode":"multi","entries":{"root":[{"kind":"file","src":"a","dest":"b"}],"labFiles":[{"kind":"file","src":"c","dest":"d"}]}}`)
	m, err = LoadAndValidateManifest(multi)
	if err != nil {
		t.Fatal(err)
	}
	if m.Mode != "multi" {
		t.Fatal("multi mode")
	}
	entries, _ := m.GetMultiEntries()
	if len(entries.Root) != 1 {
		t.Fatal("root len")
	}
	remote := map[string][]byte{"src/a/file.txt": []byte("hi"), "src/a/other.txt": []byte("hi")}
	ents := []Entry{{Kind: KindDir, Src: "src/a", Dest: "dest"}}
	expanded := ExpandDirEntries(remote, ents)
	if len(expanded) != 3 {
		t.Fatalf("expanded %d", len(expanded))
	}
}
