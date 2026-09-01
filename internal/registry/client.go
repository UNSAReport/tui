package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/UNSAReport/tui/internal/config"
)

type TemplateInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Path        string            `json:"path"`
	DistTags    map[string]string `json:"distTags,omitempty"`
	Versions    map[string]string `json:"versions,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	CachePath  string
}

func NewClient() *Client {
	base := config.GetRegistryURL()
	base = strings.TrimSuffix(base, "/")
	cache := filepath.Join(xdgCacheDir(), config.CacheFileName)
	return &Client{
		BaseURL:    base,
		HTTPClient: &http.Client{Timeout: config.RegistryTimeout},
		CachePath:  cache,
	}
}

func xdgCacheDir() string {
	if v := os.Getenv(config.EnvXDGCacheHome); v != "" {
		return filepath.Join(v, config.AppDirName)
	}
	if d, err := os.UserCacheDir(); err == nil && d != "" {
		return filepath.Join(d, config.AppDirName)
	}
	if home, _ := os.UserHomeDir(); home != "" {
		return filepath.Join(home, ".cache", config.AppDirName)
	}
	return ""
}
func (c *Client) authToken() string {
	if v := strings.TrimSpace(os.Getenv(config.EnvToken)); v != "" {
		return v
	}
	if tok := strings.TrimSpace(config.GetToken()); tok != "" {
		return tok
	}
	return ""
}

func (c *Client) setAuth(req *http.Request) {
	if tok := c.authToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}

func (c *Client) ListTemplates(ctx context.Context) ([]TemplateInfo, error) {
	if c.BaseURL == "" || strings.Contains(c.BaseURL, "github") {
		return nil, fmt.Errorf("registry unavailable: no registry URL configured")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s%s?limit=%d", c.BaseURL, config.RegistryPackagesPath, config.DefaultRegistryLimit), nil)
	if err != nil {
		return nil, fmt.Errorf("registry unavailable: %w", err)
	}
	req.Header.Set("User-Agent", "unsarep-tui")
	c.setAuth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("registry unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("registry unavailable: status %d", resp.StatusCode)
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("registry unavailable: decode %w", err)
	}
	b, _ := json.Marshal(raw)
	var p2 struct {
		Packages []TemplateInfo `json:"packages"`
		Total    int            `json:"total"`
	}
	if err2 := json.Unmarshal(b, &p2); err2 == nil && len(p2.Packages) > 0 {
		_ = os.MkdirAll(filepath.Dir(c.CachePath), config.PermDirPrivate)
		_ = os.WriteFile(c.CachePath, b, config.PermFilePrivate)
		return p2.Packages, nil
	}
	if pkgs, ok := raw["packages"].([]any); ok {
		var out []TemplateInfo
		for _, pa := range pkgs {
			if m, ok := pa.(map[string]any); ok {
				ti := TemplateInfo{}
				if v, ok := m["name"].(string); ok {
					ti.Name = v
				}
				if v, ok := m["description"].(string); ok {
					ti.Description = v
				}
				if v, ok := m["displayName"].(string); ok && ti.Description == "" {
					ti.Description = v
				}
				out = append(out, ti)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("registry unavailable: empty packages")
}


func (c *Client) GetTemplate(ctx context.Context, name string) (TemplateInfo, error) {
	name = strings.ToLower(name)
	if c.BaseURL == "" || strings.Contains(c.BaseURL, "github") {
		return TemplateInfo{}, fmt.Errorf("registry unavailable: no registry URL configured")
	}
	u := fmt.Sprintf("%s/v1/packages/%s", c.BaseURL, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return TemplateInfo{}, fmt.Errorf("registry unavailable: %w", err)
	}
	req.Header.Set("User-Agent", "unsarep-tui")
	c.setAuth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return TemplateInfo{}, fmt.Errorf("registry unavailable: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		var pkg TemplateInfo
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return TemplateInfo{}, fmt.Errorf("registry unavailable: decode %w", err)
		}
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &pkg)
		if pkg.Name == "" {
			if v, ok := raw["name"].(string); ok {
				pkg.Name = v
			}
		}
		if pkg.Description == "" {
			if v, ok := raw["description"].(string); ok {
				pkg.Description = v
			}
		}
		if vs, ok := raw["versions"].([]any); ok {
			m := make(map[string]string)
			for _, v := range vs {
				if s, ok := v.(string); ok {
					m[s] = s
				}
			}
			pkg.Versions = m
		}
		if pkg.Name != "" {
			return pkg, nil
		}
		return TemplateInfo{}, fmt.Errorf("registry unavailable: empty template name")
	case 404:
		return TemplateInfo{}, fmt.Errorf("template %q not found", name)
	default:
		return TemplateInfo{}, fmt.Errorf("registry unavailable: status %d", resp.StatusCode)
	}
}

func (c *Client) GetTemplateVersion(ctx context.Context, name, rangeSpec string) (TemplateInfo, error) {
	info, err := c.GetTemplate(ctx, name)
	if err != nil {
		return TemplateInfo{}, err
	}
	var versionsMap map[string]*semver.Version
	var distTags map[string]*semver.Version

	versionsMap = make(map[string]*semver.Version)
	for ver := range info.Versions {
		if v, err := semver.NewVersion(ver); err == nil {
			versionsMap[ver] = v
		}
	}
	distTags = make(map[string]*semver.Version)
	for tag, verStr := range info.DistTags {
		if v, err := semver.NewVersion(verStr); err == nil {
			distTags[tag] = v
		}
	}
	if len(versionsMap) == 0 && c.BaseURL != "" {
		u := fmt.Sprintf("%s/v1/packages/%s/versions", c.BaseURL, url.PathEscape(strings.ToLower(name)))
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err == nil {
			req.Header.Set("User-Agent", "unsarep-tui")
			c.setAuth(req)
			resp, err := c.HTTPClient.Do(req)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == 200 {
					var payload struct {
						Versions []struct {
							Version string `json:"version"`
						} `json:"versions"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
						for _, v := range payload.Versions {
							if sv, err := semver.NewVersion(v.Version); err == nil {
								versionsMap[v.Version] = sv
							}
						}
					}
				}
			}
		}
	}
	if len(versionsMap) == 0 && info.Version != "" {
		if v, err := semver.NewVersion(info.Version); err == nil {
			versionsMap[info.Version] = v
			if _, ok := distTags["latest"]; !ok {
				distTags["latest"] = v
			}
		}
	}

	resolved, err := resolveVersionFromMap(versionsMap, distTags, rangeSpec)
	if err != nil {
		return TemplateInfo{}, err
	}
	info.Version = resolved.Original()
	return info, nil
}

func resolveVersionFromMap(available map[string]*semver.Version, distTags map[string]*semver.Version, rangeSpec string) (*semver.Version, error) {
	switch rangeSpec {
	case "latest", "":
		if latest, ok := distTags["latest"]; ok {
			return latest, nil
		}
		return nil, fmt.Errorf("no 'latest' dist-tag found")
	case "*":
		return resolveLatest(available)
	}
	constraint, err := semver.NewConstraint(rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("invalid version range %q: %w", rangeSpec, err)
	}
	var candidates []*semver.Version
	for _, v := range available {
		if ok, _ := constraint.Validate(v); ok {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no version matching %q", rangeSpec)
	}
	max := candidates[0]
	for _, c := range candidates[1:] {
		if c.GreaterThan(max) {
			max = c
		}
	}
	return max, nil
}

func resolveLatest(versions map[string]*semver.Version) (*semver.Version, error) {
	var all []*semver.Version
	for _, v := range versions {
		if v.Prerelease() != "" {
			continue
		}
		all = append(all, v)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no stable versions found")
	}
	max := all[0]
	for _, v := range all[1:] {
		if v.GreaterThan(max) {
			max = v
		}
	}
	return max, nil
}
