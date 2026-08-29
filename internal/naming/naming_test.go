package naming

import "testing"

func TestApplyTemplate(t *testing.T) {
	if got := ApplyTemplate("{output_type}_{lab_number}", map[string]string{"lab_number": "01"}, "Informe"); got != "Informe_01" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeFilename("a/b:c"); got != "a-b-c" {
		t.Fatalf("sanitize %q", got)
	}
	if got := ApplyTemplate("{output_type}_{lab_number}_{members}", map[string]string{"lab_number": "01", "members": "a/b"}, "Informe"); got != "Informe_01_a-b" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitize(t *testing.T) {
	tests := map[string]string{
		"hello": "hello",
		"a<b":  "a-b",
		"a:b":  "a-b",
	}
	for in, want := range tests {
		if got := SanitizeFilename(in); got != want {
			t.Fatalf("sanitize %q got %q want %q", in, got, want)
		}
	}
}
