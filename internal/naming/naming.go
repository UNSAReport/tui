package naming

import (
	"fmt"
	"strings"

	"github.com/UNSAReport/tui/internal/config"
)
var reVar = config.ReVar
var reIllegal = config.ReIllegal

func SanitizeFilename(s string) string {
	s = reIllegal.ReplaceAllString(s, "-")
	// Trim trailing dots and spaces (Windows)
	s = strings.TrimRight(s, ". ")
	// Handle reserved names (Windows)
	upper := strings.ToUpper(s)
	reserved := map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
	if reserved[upper] {
		s = s + "_"
	}
	// Enforce max 255 length
	if len(s) > 255 {
		s = s[:255]
	}
	if s == "" {
		s = "_"
	}
	return s
}

func ApplyTemplate(tpl string, vars map[string]string, outputType string) (string, error) {
	var firstErr error
	res := reVar.ReplaceAllStringFunc(tpl, func(m string) string {
		sub := reVar.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		key := sub[1]
		if key == "output_type" {
			return SanitizeFilename(outputType)
		}
		if v, ok := vars[key]; ok {
			return SanitizeFilename(v)
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("unknown template variable %q", key)
		}
		return m
	})
	if firstErr != nil {
		return "", firstErr
	}
	return res, nil
}
