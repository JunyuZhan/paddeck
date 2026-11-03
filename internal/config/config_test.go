package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsAndValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  listen: ":9080"
collector:
  interval: 5s
hosts:
  - id: web-01
    address: 192.168.1.10
    user: youruser
services:
  - name: NAS
    url: http://192.168.1.5/
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	// empty means "let the SSH layer auto-discover ~/.ssh keys or ssh-agent"
	if cfg.Collector.SSHKey != "" {
		t.Errorf("ssh key default = %q, want empty (auto)", cfg.Collector.SSHKey)
	}
	if cfg.Hosts[0].Port != 22 {
		t.Errorf("port default = %d", cfg.Hosts[0].Port)
	}
	if cfg.Hosts[0].DockerPath != "docker" {
		t.Errorf("docker path default = %q", cfg.Hosts[0].DockerPath)
	}
	if cfg.Collector.DockerInterval.Std().Seconds() != 30 {
		t.Errorf("docker interval default = %v", cfg.Collector.DockerInterval)
	}
}

func TestLoadRejectsDuplicateHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
hosts:
  - id: a
    address: 1.2.3.4
    user: root
  - id: a
    address: 1.2.3.5
    user: root
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestExpandEnv(t *testing.T) {
	t.Setenv("PADDECK_TEST_VALUE", "secret")
	cases := map[string]string{
		"a ${PADDECK_TEST_VALUE} b":         "a secret b",
		"${PADDECK_TEST_MISSING:-fallback}": "fallback",
		"${PADDECK_TEST_MISSING}":           "",
		"plain":                             "plain",
		"$PADDECK_TEST_VALUE":               "$PADDECK_TEST_VALUE", // only ${} is expanded
	}
	for in, want := range cases {
		if got := expandEnv(in); got != want {
			t.Errorf("expandEnv(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadExpandsEnvironment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
hosts:
  - id: router
    type: ikuai
    address: 192.168.1.1
    username: admin
    password: ${PADDECK_TEST_PW}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PADDECK_TEST_PW", "hunter2")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hosts[0].Password != "hunter2" {
		t.Errorf("password = %q", cfg.Hosts[0].Password)
	}
}

func TestWriteStarter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// first write creates a valid, loadable config
	if err := WriteStarter(path, false); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("starter config must load: %v", err)
	}
	// second write refuses to clobber
	if err := WriteStarter(path, false); err == nil {
		t.Fatal("expected refusal to overwrite")
	}
	// unless forced
	if err := WriteStarter(path, true); err != nil {
		t.Fatalf("forced write: %v", err)
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("hosts: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(path)
	if err != nil || got != path {
		t.Fatalf("Resolve(explicit) = %q, %v", got, err)
	}
	if _, err := Resolve(filepath.Join(dir, "nope.yaml")); err == nil {
		t.Fatal("expected error for missing explicit path")
	}
}
