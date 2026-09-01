package auth

import (
	"errors"
	"os"
	"os/exec"
	"runtime"

	"github.com/UNSAReport/tui/internal/config"
)

var ErrNoBrowser = errors.New("no browser available")

func OpenBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return ErrNoBrowser
	}
	return nil
}

func IsHeadless() bool {
	if os.Getenv(config.EnvSSHConnection) != "" {
		return true
	}
	if runtime.GOOS == "linux" && os.Getenv(config.EnvDisplay) == "" && os.Getenv(config.EnvWaylandDisplay) == "" {
		return true
	}
	if runtime.GOOS == "windows" && os.Getenv("TERM") == "" {
		return true
	}
	return false
}
