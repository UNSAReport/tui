package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/UNSAReport/tui/internal/auth"
	"github.com/UNSAReport/tui/internal/config"
	"github.com/UNSAReport/tui/internal/i18n"
	"github.com/UNSAReport/tui/internal/tui"
)

var version = config.Version

func main() {
	i18n.Init()

	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "login":
			handleLogin(args[1:])
			return
		case "logout":
			handleLogout(args[1:])
			return
		case "whoami", "status", "auth":
			if args[0] == "auth" && len(args) > 1 && args[1] == "status" {
				handleWhoami(args[2:])
				return
			}
			if args[0] == "whoami" || args[0] == "status" {
				handleWhoami(args[1:])
				return
			}
			if args[0] == "auth" {
				fmt.Fprintln(os.Stderr, "usage: unsarep auth status")
				os.Exit(1)
			}
		case "help", "--help", "-h":
			printHelp()
			os.Exit(0)
		case "version", "--version", "-v":
			fmt.Printf("unsarep %s\n", version)
			os.Exit(0)
		default:
			if !strings.HasPrefix(args[0], "-") {
				fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
				printHelp()
				os.Exit(2)
			}
		}
	}

	var showHelp bool
	var showVersion bool
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}
	if showVersion {
		fmt.Printf("unsarep %s\n", version)
		os.Exit(0)
	}
	if len(flag.Args()) > 0 && (flag.Arg(0) == "help" || flag.Arg(0) == "--help") {
		printHelp()
		os.Exit(0)
	}

	m := tui.NewRootModel(tui.RootOptions{})
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func handleLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	noBrowser := fs.Bool("no-browser", false, "Don't open browser, print URL only")
	tokenFlag := fs.String("token", "", "PAT token (unsareport_pat_...)")
	fs.BoolVar(noBrowser, "n", false, "Don't open browser (shorthand)")
	_ = fs.Parse(args)

	ctx := context.Background()
	client := auth.NewClient()

	if *tokenFlag != "" {
		cred, err := client.LoginWithToken(ctx, *tokenFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Logged in as %s <%s>\n", cred.Name, cred.Email)
		if cred.Email == "" {
			fmt.Printf("Token stored (%s...)\n", cred.PAT[:min(10, len(cred.PAT))])
		}
		return
	}
	cred, err := client.Login(ctx, *noBrowser)
	if err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Logged in as %s <%s>\n", cred.Name, cred.Email)
	if cred.Email == "" && cred.UserID != "" {
		fmt.Printf("Logged in as %s\n", cred.UserID)
	}
}

func handleLogout(args []string) {
	_ = args
	client := auth.NewClient()
	if err := client.Logout(); err != nil {
		fmt.Fprintf(os.Stderr, "logout failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Logged out")
}

func handleWhoami(args []string) {
	fs := flag.NewFlagSet("whoami", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "JSON output")
	_ = fs.Parse(args)
	ctx := context.Background()
	client := auth.NewClient()
	cred, user, err := client.Status(ctx)
	if err != nil {
		if cred != nil && strings.Contains(err.Error(), "offline") {
			if *jsonOut {
				b, _ := json.MarshalIndent(cred, "", "  ")
				fmt.Println(string(b))
				return
			}
			fmt.Printf("Logged in (offline) — token %s...\n", cred.PAT[:min(10, len(cred.PAT))])
			if cred.Email != "" {
				fmt.Printf("Email: %s\nName: %s\n", cred.Email, cred.Name)
			}
			return
		}
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if *jsonOut {
		out := map[string]any{"credentials": cred, "user": user}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return
	}
	if user != nil {
		fmt.Printf("Logged in as %s <%s>\n", user.Name, user.Email)
		if len(user.Roles) > 0 {
			fmt.Printf("Roles: %v\n", user.Roles)
		}
	} else if cred != nil {
		fmt.Printf("Logged in as %s <%s>\n", cred.Name, cred.Email)
	}
	if cred != nil && cred.PAT != "" {
		fmt.Printf("PAT: %s...\n", cred.PAT[:min(10, len(cred.PAT))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func printHelp() {
	fmt.Println(`unsarep — UNSAReport TUI (Bubble Tea)

Usage:
  unsarep [command] [flags]
  unsarep [flags]          # launch TUI

Commands:
  login [--no-browser] [--token <PAT>]   Browser login via website provider picker (loopback 127.0.0.1)
  logout                                 Clear stored credentials
  whoami [--json]                         Show authenticated user (also: auth status, status)
  help                                   Show help
  version                                Show version

Flags:
  -h, --help      Show help
  -v, --version   Show version

Auth:
  Credentials stored in OS keychain (0600 file at $XDG_CONFIG_HOME/unsareport/credentials.json when keychain unavailable).
  Env override: UNSAREP_TOKEN (transient, not persisted), UNSAREP_IDP_ISSUER, UNSAREP_REGISTRY_URL.
  Browser flow: opens website /auth/login?tui_callback=http://127.0.0.1:<port>/callback&state=...
  Headless: prints URL and waits for callback; if no browser, paste PAT via --token.

Navigation (TUI):
  1-4         Switch apps (Registry, Docs, Auth, Slides)
  h/l         Prev/next app
  j/k, up/down  Navigate sidebar
  enter       Select function
  tab         Toggle focus sidebar/main
  q, ctrl+c   Quit
  ?           Help overlay
  r           Retry
  b, esc      Back

Project-centric: detects unsareport.json via walk-up and preselects project context.`)
}
