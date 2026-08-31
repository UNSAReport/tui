package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/UNSAReport/tui/internal/config"
)

//go:embed locales/*.json
var localesFS embed.FS

var current = "en"
var tables = map[string]map[string]string{}

func Init() {
	for _, lang := range []string{"en", "es"} {
		b, err := localesFS.ReadFile("locales/" + lang + ".json")
		if err != nil {
			continue
		}
		var m map[string]string
		if err := json.Unmarshal(b, &m); err == nil {
			tables[lang] = m
		}
	}
	locale := config.GetLocale()
	if locale == "" {
		locale = detectEnvLocale()
	}
	locale = strings.ToLower(locale)
	if strings.HasPrefix(locale, "es") {
		current = "es"
	} else {
		current = "en"
	}
}

func detectEnvLocale() string {
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" {
			v = strings.ToLower(v)
			if strings.HasPrefix(v, "es") {
				return "es"
			}
			if strings.HasPrefix(v, "en") {
				return "en"
			}
			return v
		}
	}
	return "en"
}

func T(key string, args ...any) string {
	if m, ok := tables[current]; ok {
		if v, ok := m[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(v, args...)
			}
			return v
		}
	}
	if m, ok := tables["en"]; ok {
		if v, ok := m[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(v, args...)
			}
			return v
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(key, args...)
	}
	return key
}

func Current() string { return current }

func SetLocale(l string) {
	if strings.HasPrefix(strings.ToLower(l), "es") {
		current = "es"
	} else {
		current = "en"
	}
}
