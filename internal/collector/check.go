package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JunyuZhan/paddeck/internal/config"
)

// CheckResult is one line of the `paddeck -check` report.
type CheckResult struct {
	Group  string // host | pve | service
	Name   string
	Target string
	OK     bool
	Detail string
	Hint   string
}

// Check probes every configured target once with short timeouts and returns a
// report. It exists so a new user can validate a config before starting the
// panel instead of staring at a dashboard full of "OFFLINE".
func Check(ctx context.Context, cfg *config.Config) []CheckResult {
	sshTimeout := cfg.Collector.SSHTimeout.Std()
	if sshTimeout <= 0 {
		sshTimeout = 8 * time.Second
	}
	httpTimeout := cfg.Collector.HTTPTimeout.Std()
	if httpTimeout <= 0 {
		httpTimeout = 3 * time.Second
	}

	var out []CheckResult
	for _, h := range cfg.Hosts {
		out = append(out, checkHost(ctx, h, cfg, sshTimeout))
	}
	for _, p := range cfg.PVE {
		out = append(out, checkPVE(ctx, p, sshTimeout))
	}
	prober := NewProber(httpTimeout)
	for _, svc := range cfg.Services {
		cctx, cancel := context.WithTimeout(ctx, httpTimeout)
		st := prober.Probe(cctx, svc.Name, svc.URL)
		cancel()
		res := CheckResult{Group: "service", Name: svc.Name, Target: svc.URL, OK: st.OK}
		if st.OK {
			res.Detail = fmt.Sprintf("HTTP %d in %dms", st.Code, st.LatencyMS)
		} else {
			res.Detail = fmt.Sprintf("HTTP %d %s", st.Code, strings.TrimSpace(st.Error))
			res.Hint = "check the URL, DNS and that the service allows requests from this host"
		}
		out = append(out, res)
	}
	return out
}

func checkHost(ctx context.Context, h config.Host, cfg *config.Config, timeout time.Duration) CheckResult {
	cctx, cancel := context.WithTimeout(ctx, timeout+2*time.Second)
	defer cancel()

	res := CheckResult{Group: "host", Name: h.Name, Target: h.Address}
	if h.IsIKuai() {
		c := NewIKuaiCollector(h.APIURL, h.Username, h.Password, timeout)
		m := c.Collect(cctx, h.ID)
		res.OK = m.Online
		if m.Online {
			res.Detail = fmt.Sprintf("API ok · CPU %s · %d clients", pctText(m.CPU), m.Clients)
		} else {
			res.Detail = m.Err
			res.Hint = "check api_url / username / password (the API listens on :444 by default)"
		}
		return res
	}

	ssh := &SSH{Addr: h.Address, User: h.User, Port: h.Port, KeyPath: h.SSHKey, Timeout: timeout}
	kind := "linux"
	script := `uname -s 2>/dev/null; uname -r 2>/dev/null`
	if h.Type == "macos" {
		kind = "macos"
		script = `uname -s 2>/dev/null; sw_vers -productVersion 2>/dev/null`
	}
	outStr, err := ssh.Run(cctx, script)
	if err != nil {
		res.Detail = strings.TrimSpace(err.Error())
		res.Hint = fmt.Sprintf("ssh %s@%s — check address, user and that your key is in ~/.ssh/authorized_keys", h.User, h.Address)
		return res
	}
	parts := strings.Fields(outStr)
	res.OK = true
	res.Detail = kind + " " + strings.Join(parts, " ")

	if h.Docker {
		dc := NewDockerCollector(ssh, h.DockerPath, h.DockerSudo)
		dctx, dcancel := context.WithTimeout(ctx, timeout+4*time.Second)
		list, err := dc.List(dctx)
		dcancel()
		switch {
		case err != nil:
			res.OK = false
			res.Detail += " · docker: " + strings.TrimSpace(err.Error())
			res.Hint = "docker is enabled for this host: check the docker binary path" +
				" (docker_path) and whether sudo is required (docker_sudo)"
		default:
			res.Detail += fmt.Sprintf(" · docker ok (%d containers)", len(list))
		}
	}
	return res
}

func checkPVE(ctx context.Context, p config.PVEHost, timeout time.Duration) CheckResult {
	cctx, cancel := context.WithTimeout(ctx, timeout+6*time.Second)
	defer cancel()

	res := CheckResult{Group: "pve", Name: p.Name, Target: p.Address}
	ssh := &SSH{Addr: p.Address, User: p.User, Port: p.Port, KeyPath: p.SSHKey, Timeout: timeout}
	pc := NewPVECollector(ssh, p.Name)
	nodes, guests, err := pc.Collect(cctx)
	if err != nil {
		res.Detail = strings.TrimSpace(err.Error())
		res.Hint = "PVE needs root SSH and pvesh on the node (Proxmox VE 6+)"
		return res
	}
	res.OK = len(nodes) > 0
	res.Detail = fmt.Sprintf("%d node(s), %d guest(s)", len(nodes), len(guests))
	if !res.OK {
		res.Detail = "pvesh returned no nodes"
	}
	return res
}

func pctText(v float64) string {
	if v >= 10 {
		return fmt.Sprintf("%.0f%%", v)
	}
	return fmt.Sprintf("%.1f%%", v)
}
