package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Container is one Docker container.
type Container struct {
	Host     string  `json:"host"`
	Name     string  `json:"name"`
	Image    string  `json:"image"`
	State    string  `json:"state"`
	Status   string  `json:"status"`
	CPU      float64 `json:"cpu"`
	Mem      uint64  `json:"mem"`
	MemLimit uint64  `json:"mem_limit"`
	Restarts int     `json:"restarts"`
	HasStats bool    `json:"has_stats"`
}

type dockerPS struct {
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

type dockerStat struct {
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemUsage string `json:"MemUsage"`
}

// DockerCollector lists containers and optionally reads their live stats.
type DockerCollector struct {
	ssh  *SSH
	path string
	sudo bool
}

func NewDockerCollector(ssh *SSH, path string, sudo bool) *DockerCollector {
	if path == "" {
		path = "docker"
	}
	return &DockerCollector{ssh: ssh, path: path, sudo: sudo}
}

func (d *DockerCollector) cmd(args string) string {
	if d.sudo {
		return fmt.Sprintf("sudo -n %s %s", d.path, args)
	}
	return fmt.Sprintf("%s %s", d.path, args)
}

func (d *DockerCollector) innerPS() string {
	if d.sudo {
		return fmt.Sprintf("sudo -n %s ps -aq", d.path)
	}
	return fmt.Sprintf("%s ps -aq", d.path)
}

// List returns all containers without live stats.
func (d *DockerCollector) List(ctx context.Context) ([]Container, error) {
	out, err := d.ssh.Run(ctx, d.cmd(`ps -a --format '{{json .}}'`))
	if err != nil {
		return nil, err
	}
	var result []Container
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var p dockerPS
		if err := json.Unmarshal([]byte(line), &p); err != nil {
			continue
		}
		state := p.State
		if state == "" {
			if strings.HasPrefix(p.Status, "Up") {
				state = "running"
			} else {
				state = "exited"
			}
		}
		result = append(result, Container{
			Name:   strings.TrimPrefix(p.Names, "/"),
			Image:  p.Image,
			State:  state,
			Status: p.Status,
		})
	}
	return result, nil
}

// Stats reads CPU/memory for running containers plus restart counts.
func (d *DockerCollector) Stats(ctx context.Context) (map[string]Container, error) {
	result := map[string]Container{}

	out, err := d.ssh.Run(ctx, d.cmd(`stats --no-stream --format '{{json .}}'`))
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var s dockerStat
			if err := json.Unmarshal([]byte(line), &s); err != nil {
				continue
			}
			used, limit := parseMemUsage(s.MemUsage)
			result[s.Name] = Container{
				Name:     s.Name,
				CPU:      parsePercent(s.CPUPerc),
				Mem:      used,
				MemLimit: limit,
				HasStats: true,
			}
		}
	}

	out, err = d.ssh.Run(ctx, d.cmd(`inspect --format '{{.Name}}|{{.RestartCount}}' $(`+d.innerPS()+`) 2>/dev/null`))
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.Split(strings.TrimSpace(line), "|")
			if len(parts) != 2 {
				continue
			}
			name := strings.TrimPrefix(parts[0], "/")
			n, _ := strconv.Atoi(parts[1])
			c := result[name]
			c.Name = name
			c.Restarts = n
			result[name] = c
		}
	}
	return result, nil
}

func parsePercent(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	return round2(v)
}

// parseMemUsage parses Docker's "1.5MiB / 7.8GiB" format.
func parseMemUsage(s string) (used, limit uint64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseHumanBytes(parts[0]), parseHumanBytes(parts[1])
}

func parseHumanBytes(s string) uint64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0B" {
		return 0
	}
	units := []struct {
		suffix string
		mult   float64
	}{
		{"GiB", 1024 * 1024 * 1024},
		{"MiB", 1024 * 1024},
		{"KiB", 1024},
		{"GB", 1000 * 1000 * 1000},
		{"MB", 1000 * 1000},
		{"KB", 1000},
		{"B", 1},
	}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			num := strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			v, err := strconv.ParseFloat(num, 64)
			if err != nil {
				return 0
			}
			return uint64(v * u.mult)
		}
	}
	return 0
}
