package auth

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
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
	if os.Getenv("SSH_CONNECTION") != "" {
		return true
	}
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return true
	}
	return false
}
