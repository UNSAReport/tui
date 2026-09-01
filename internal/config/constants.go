package config

import (
	"regexp"
	"time"
)

// Default URLs and endpoints
const (
	DefaultRegistryURL = "https://registry.unsareport.org"
	DefaultAuthURL     = "https://auth.unsareport.org"
	SchemaBaseURL      = "https://raw.githubusercontent.com/UNSAReport/UNSAReport"
	DefaultCallbackHost = "127.0.0.1:0"
	CallbackBaseURLPrefix = "http://"
	CallbackPath          = "/callback"
)

// Version — set via ldflag -X github.com/UNSAReport/tui/internal/config.Version=...
var Version = "dev"

// Defaults for project config
const (
	DefaultPrompt         = "❯ "
	DefaultColumns        = 120
	DefaultRows           = 500
	DefaultSrcDir         = "src"
	DefaultSubmissionDir  = "submission"
	DefaultReportFile     = "report.typ"
	DefaultReportWord     = "Informe"
	DefaultCodeWord       = "Código Fuente"
	DefaultFileTemplate   = "{output_type}_{lab_number}"
)

// Credentials / keyring
const (
	PATPrefix      = "unsareport_pat_"
	KeyringService = "unsareport"
	KeyringUser    = "pat"
)

// Limits and timeouts
const (
	DefaultRegistryLimit = 100
)

var (
	AuthTimeout             = 10 * time.Second
	RegistryTimeout         = 30 * time.Second
	CallbackTimeout         = 5 * time.Minute
	CallbackReadHeaderTimeout = 5 * time.Second
)

// File permissions
const (
	PermDirPrivate  = 0o700
	PermFilePrivate = 0o600
	PermDirPublic   = 0o755
	PermFilePublic  = 0o644
)

// Env var names — canonical
const (
	EnvRegistryURL     = "UNSAREP_REGISTRY_URL"
	EnvIDPIssuer       = "UNSAREP_IDP_ISSUER"
	EnvToken           = "UNSAREP_TOKEN"
	EnvTokenPath       = "UNSAREP_TOKEN_PATH"
	EnvCredentialsPath = "UNSAREP_CREDENTIALS_PATH"
	EnvWebsiteURL      = "UNSAREP_WEBSITE_URL"
	EnvDest            = "UNSAREP_DEST"
	EnvSession         = "UNSAREP_SESSION"
	EnvLocal           = "UNSAREP_LOCAL"
	EnvFreezeFlags     = "UNSAREP_FREEZE_FLAGS"
	EnvLocale          = "UNSAREP_LOCALE"
	EnvXDGConfigHome   = "XDG_CONFIG_HOME"
	EnvXDGCacheHome    = "XDG_CACHE_HOME"
	EnvSSHConnection   = "SSH_CONNECTION"
	EnvDisplay         = "DISPLAY"
	EnvWaylandDisplay  = "WAYLAND_DISPLAY"
	EnvLang            = "LANG"
	EnvLCAll           = "LC_ALL"
	EnvLCMessages      = "LC_MESSAGES"
)

// XDG / file layout
const (
	AppDirName           = "unsareport"
	ConfigFileName       = "unsareport.json"
	XDGConfigFileName    = "config.json"
	CredentialsFileName  = "credentials.json"
	TokenFileName        = "token"
	CacheFileName        = "registry.json"
	LockFileName         = ".unsareport.lock"
)

// Regexes (compiled)
var (
	ReVar     = regexp.MustCompile(`\{(\w+)\}`)
	ReIllegal = regexp.MustCompile(`[<>:"/\\|?*]`)
)

// TUI keys
const (
	Key1     = "1"
	Key2     = "2"
	Key3     = "3"
	Key4     = "4"
	KeyH     = "h"
	KeyL     = "l"
	KeyR     = "r"
	KeyUp    = "up"
	KeyDown  = "down"
	KeyEnter = "enter"
)

// Colors defaults
const (
	ColorPrompt  = "32"
	ColorCommand = "36"
	ColorArgs    = "33"
	ColorReset   = "0"
)

// Registry pagination
const RegistryPackagesPath = "/v1/packages"

// EnvNames exposes canonical env vars for docs/tests
var EnvNames = []string{
	EnvRegistryURL,
	EnvIDPIssuer,
	EnvToken,
	EnvTokenPath,
	EnvCredentialsPath,
	EnvWebsiteURL,
	EnvDest,
	EnvSession,
	EnvLocal,
	EnvFreezeFlags,
}
