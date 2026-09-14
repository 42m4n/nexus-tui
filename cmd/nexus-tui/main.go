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

var version = "dev"

// buildClients makes one client per profile that has a url.
func buildClients(cfg config.Config, writes bool) (map[string]*nexus.Client, error) {
	clients := map[string]*nexus.Client{}
	for name, p := range cfg.Profiles {
		if p.URL == "" {
			continue
		}
		c, err := nexus.New(p.URL, p.Insecure, p.Username, p.Password, p.Token)
		if err != nil {
			return nil, fmt.Errorf("profile %s: %w", name, err)
		}
		c.Writes = writes
		clients[name] = c
	}
	return clients, nil
}

func main() {
	profile := flag.String("profile", "", "config profile name")
	cfgPath := flag.String("config", "", "config file path (default ~/.config/nexus-tui/config.yaml)")
	insecure := flag.Bool("insecure", false, "skip TLS certificate verification")
	allowWrites := flag.Bool("allow-writes", false, "enable gated actions (delete repo/user, invalidate cache)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("nexus-tui", version)
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if *insecure {
		for name, p := range cfg.Profiles {
			p.Insecure = true
			cfg.Profiles[name] = p
		}
	}
	clients, err := buildClients(cfg, *allowWrites)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	cur := *profile
	if cur == "" {
		cur = cfg.Current
	}
	if cur == "" {
		cur = "env"
	}
	if _, ok := clients[cur]; !ok {
		fmt.Fprintln(os.Stderr, "config: no url set; export NEXUS_URL or set url in config")
		os.Exit(1)
	}

	m := ui.New(clients, cur)
	ui.Version = version
	prog := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
