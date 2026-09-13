package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"nexus-tui/internal/config"
	"nexus-tui/internal/nexus"
	"nexus-tui/internal/ui"
)

func main() {
	profile := flag.String("profile", "", "config profile name")
	cfgPath := flag.String("config", "", "config file path (default ~/.config/nexus-tui/config.yaml)")
	insecure := flag.Bool("insecure", false, "skip TLS certificate verification")
	allowWrites := flag.Bool("allow-writes", false, "enable destructive actions (delete repo/user)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("nexus-tui 0.1.0")
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	p, err := cfg.Select(*profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if p.URL == "" {
		fmt.Fprintln(os.Stderr, "config: no url set; export NEXUS_URL or set url in config")
		os.Exit(1)
	}
	if *insecure {
		p.Insecure = true
	}

	c, err := nexus.New(p.URL, p.Insecure, p.Username, p.Password, p.Token)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c.Writes = *allowWrites

	m := ui.New(c)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
