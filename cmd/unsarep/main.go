package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/UNSAReport/tui/internal/i18n"
	"github.com/UNSAReport/tui/internal/tui"
)

var version = "dev"
func main() {
	i18n.Init()
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
	// Also handle --help as arg fallback for go run without flag package? flag already covers.
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

func printHelp() {
	fmt.Println(`unsarep — UNSAReport TUI (Bubble Tea)

Usage:
  unsarep [flags]

Flags:
  -h, --help      Show help
  -v, --version   Show version

Navigation:
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
