package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration loaded from YAML.
type Config struct {
	Server    Server    `yaml:"server"`
	Collector Collector `yaml:"collector"`
	Hosts     []Host    `yaml:"hosts"`
	PVE       []PVEHost `yaml:"pve"`
	Services  []Service `yaml:"services"`
}

type Server struct {
	Listen   string `yaml:"listen"`
	BasePath string `yaml:"base_path"`
}

type Collector struct {
	Interval        Duration `yaml:"interval"`
	DockerInterval  Duration `yaml:"docker_interval"`
	HistoryInterval Duration `yaml:"history_interval"`
	HTTPTimeout     Duration `yaml:"http_timeout"`
	SSHTimeout      Duration `yaml:"ssh_timeout"`
	RetentionDays   int      `yaml:"retention_days"`
	SSHKey          string   `yaml:"ssh_key"`
}

// Host describes one monitored machine.
//
// Type selects the collector: "linux" (default, over SSH) or "ikuai"
// (iKuai router, over its HTTP API).
type Host struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Address     string   `yaml:"address"`
	User        string   `yaml:"user"`
	Port        int      `yaml:"port"`
	SSHKey      string   `yaml:"ssh_key"`
	APIURL      string   `yaml:"api_url"`
	Username    string   `yaml:"username"`
	Password    string   `yaml:"password"`
	Docker      bool     `yaml:"docker"`
	DockerSudo  bool     `yaml:"docker_sudo"`
	DockerPath  string   `yaml:"docker_path"`
	DiskMounts  []string `yaml:"disk_mounts"`
	PVE         bool     `yaml:"pve"`
	Labels      []string `yaml:"labels"`
	Description string   `yaml:"description"`
}

// IsIKuai reports whether the host is collected through the iKuai API.
func (h Host) IsIKuai() bool { return h.Type == "ikuai" }

// PVEHost describes a Proxmox VE node reachable over SSH.
type PVEHost struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Address string `yaml:"address"`
	User    string `yaml:"user"`
	Port    int    `yaml:"port"`
	SSHKey  string `yaml:"ssh_key"`
}

// Service is an HTTP endpoint to probe.
type Service struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

// Duration wraps time.Duration so YAML can use "10s".
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) Std() time.Duration { return time.Duration(d) }

// Defaults fills unset values with sensible defaults.
func (c *Config) Defaults() {
	if c.Server.Listen == "" {
		c.Server.Listen = ":9080"
	}
	if c.Collector.Interval == 0 {
		c.Collector.Interval = Duration(10 * time.Second)
	}
	if c.Collector.DockerInterval == 0 {
		c.Collector.DockerInterval = Duration(30 * time.Second)
	}
	if c.Collector.HistoryInterval == 0 {
		c.Collector.HistoryInterval = Duration(60 * time.Second)
	}
	if c.Collector.HTTPTimeout == 0 {
		c.Collector.HTTPTimeout = Duration(3 * time.Second)
	}
	if c.Collector.SSHTimeout == 0 {
		c.Collector.SSHTimeout = Duration(8 * time.Second)
	}
	if c.Collector.RetentionDays == 0 {
		c.Collector.RetentionDays = 30
	}
	// Empty means "let the SSH layer pick": ssh_key in the config, then
	// ~/.ssh/id_ed25519, then ~/.ssh/id_rsa, then ssh-agent.
	for i := range c.Hosts {
		h := &c.Hosts[i]
		if h.Type == "" {
			h.Type = "linux"
		}
		if h.Port == 0 {
			h.Port = 22
		}
		if h.Name == "" {
			h.Name = h.ID
		}
		if h.DockerPath == "" {
			h.DockerPath = "docker"
		}
		if h.SSHKey == "" {
			h.SSHKey = c.Collector.SSHKey
		}
		if h.IsIKuai() && h.APIURL == "" {
			h.APIURL = "https://" + h.Address + ":444"
		}
	}
	for i := range c.PVE {
		if c.PVE[i].Port == 0 {
			c.PVE[i].Port = 22
		}
		if c.PVE[i].Name == "" {
			c.PVE[i].Name = c.PVE[i].ID
		}
		if c.PVE[i].SSHKey == "" {
			c.PVE[i].SSHKey = c.Collector.SSHKey
		}
	}
}

// Validate checks required fields and uniqueness.
func (c *Config) Validate() error {
	seen := map[string]bool{}
	for _, h := range c.Hosts {
		if h.ID == "" {
			return fmt.Errorf("hosts: id is required")
		}
		if seen[h.ID] {
			return fmt.Errorf("hosts: duplicate id %q", h.ID)
		}
		seen[h.ID] = true
		if h.Address == "" {
			return fmt.Errorf("hosts[%s]: address is required", h.ID)
		}
		switch h.Type {
		case "linux", "macos":
			if h.User == "" {
				return fmt.Errorf("hosts[%s]: user is required", h.ID)
			}
		case "ikuai":
			if h.Username == "" || h.Password == "" {
				return fmt.Errorf("hosts[%s]: username and password are required for type ikuai", h.ID)
			}
			if h.APIURL == "" {
				return fmt.Errorf("hosts[%s]: api_url is required for type ikuai", h.ID)
			}
		default:
			return fmt.Errorf("hosts[%s]: unknown type %q", h.ID, h.Type)
		}
	}
	for _, p := range c.PVE {
		if p.ID == "" {
			return fmt.Errorf("pve: id is required")
		}
		if seen[p.ID] {
			return fmt.Errorf("pve: duplicate id %q", p.ID)
		}
		seen[p.ID] = true
		if p.Address == "" {
			return fmt.Errorf("pve[%s]: address is required", p.ID)
		}
	}
	for _, s := range c.Services {
		if s.Name == "" || s.URL == "" {
			return fmt.Errorf("services: name and url are required")
		}
	}
	return nil
}

// Load reads, defaults and validates a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal([]byte(expandEnv(string(data))), &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.Defaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// ---------------------------------------------------------------- discovery

// Resolve finds the config file. An explicit path always wins; otherwise the
// usual locations are tried so `paddeck` works with no flags in most setups.
func Resolve(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config %q: %w", explicit, err)
		}
		return explicit, nil
	}
	if p := os.Getenv("PADDECK_CONFIG"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	home, _ := os.UserHomeDir()
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" && home != "" {
		xdg = filepath.Join(home, ".config")
	}
	candidates := []string{
		"config.yaml",
		"config.local.yaml",
		filepath.Join(xdg, "paddeck", "config.yaml"),
		"/config/config.yaml",
	}
	tried := make([]string, 0, len(candidates))
	for _, p := range candidates {
		tried = append(tried, p)
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no config file found (looked in: %v)\n"+
		"\n  Create one:        paddeck -init\n"+
		"  Or point at one:   paddeck -config /path/to/config.yaml\n"+
		"  Or set:            PADDECK_CONFIG=/path/to/config.yaml", tried)
}

// ---------------------------------------------------------------- env vars

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

// expandEnv replaces ${VAR} and ${VAR:-default} so secrets can stay out of the
// config file (a Kubernetes/Docker convention users already know).
func expandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(m string) string {
		g := envPattern.FindStringSubmatch(m)
		if v, ok := os.LookupEnv(g[1]); ok && v != "" {
			return v
		}
		if g[2] != "" {
			return g[3]
		}
		return ""
	})
}

// ---------------------------------------------------------------- starter

// StarterConfig is what `paddeck -init` writes: the smallest config that runs,
// with every optional section spelled out as a comment.
const StarterConfig = `# PadDeck configuration - generated by ` + "`paddeck -init`" + `
#
# Everything here is optional except one of hosts/services/pve.
# Values support environment expansion: password: ${IKUAI_PASSWORD}

server:
  listen: ":9080"

collector:
  interval: 10s          # how often hosts are polled
  docker_interval: 30s   # docker stats is expensive, poll it less often
  history_interval: 60s  # one row per host per minute
  http_timeout: 3s
  ssh_timeout: 8s
  retention_days: 30
  # ssh_key: ~/.ssh/id_ed25519   # empty = auto (ssh_key -> ~/.ssh/id_ed25519 -> ~/.ssh/id_rsa -> ssh-agent)

hosts:
  - id: web-01
    name: web-01
    address: 192.168.1.10      # change me
    user: youruser             # change me
    docker: true
    # docker_sudo: true        # if docker needs sudo -n
    # docker_path: /usr/local/bin/docker
    # disk_mounts: ["/"]

# Optional: macOS hosts use the same block with type: macos
#  - id: mac
#    name: Mac
#    type: macos
#    address: 192.168.1.20
#    user: youruser

# Optional: iKuai routers are collected over their web API (no SSH)
#  - id: ikuai
#    name: iKuai
#    type: ikuai
#    address: 192.168.1.1
#    username: admin
#    password: ${IKUAI_PASSWORD}

# Optional: Proxmox VE nodes (root SSH)
#pve:
#  - id: pve
#    name: pve
#    address: 192.168.1.4
#    user: root

services:
  - name: NAS
    url: http://192.168.1.5/   # change me
`

// WriteStarter writes the starter config. It never overwrites an existing file
// unless force is set.
func WriteStarter(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (use -force to overwrite)", path)
		}
	}
	if err := os.WriteFile(path, []byte(StarterConfig), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
