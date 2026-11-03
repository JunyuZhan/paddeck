// Package demo serves a synthetic fleet so PadDeck can be evaluated (and
// screenshotted) without any real infrastructure or config file.
package demo

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/JunyuZhan/paddeck/internal/collector"
	"github.com/JunyuZhan/paddeck/internal/store"
)

// histLen is 6 hours of one-minute samples.
const histLen = 360

// Source is a fake collector + history pair used by `paddeck -demo`.
type Source struct {
	mu   sync.RWMutex
	snap collector.Snapshot
	hist map[string][]store.Sample
}

// New builds a small, believable fleet with generic names only.
func New() *Source {
	s := &Source{hist: map[string][]store.Sample{}}
	s.build()
	s.seedHistory()
	return s
}

type host struct {
	id, name, kind, addr string
	cpu, memPct          float64
	memTotal             uint64
	load, temp           float64
	up                   uint64
	kernel               string
	clients, conns       int
	wanUp, wanDown       uint64
	wanIP                string
}

func (h host) snapshot() collector.HostView {
	v := collector.HostView{
		ID: h.id, Name: h.name, Kind: h.kind, Address: h.addr,
		HostMetrics: collector.HostMetrics{
			CPU:      h.cpu,
			MemTotal: h.memTotal,
			MemUsed:  uint64(float64(h.memTotal) * h.memPct / 100),
			Load1:    h.load, Load5: h.load * 1.1, Load15: h.load * 0.9,
			Uptime: h.up, Temp: h.temp, Kernel: h.kernel, Hostname: h.name,
			Online: true, LastSeen: time.Now().Unix(),
		},
	}
	if h.kind == "ikuai" {
		v.HostMetrics.Clients = h.clients
		v.HostMetrics.Conns = h.conns
		v.HostMetrics.WANUp = h.wanUp
		v.HostMetrics.WANDown = h.wanDown
		v.HostMetrics.WANIP = h.wanIP
		v.HostMetrics.WANStatus = "up"
		v.HostMetrics.Kernel = "3.7.14"
	} else {
		v.HostMetrics.NetRx = 4200
		v.HostMetrics.NetTx = 1300
	}
	return v
}

func (s *Source) build() {
	hosts := []host{
		{"web-01", "web-01", "linux", "192.168.1.10", 18, 42, 16 << 30, 0.42, 44, 412800, "6.8.0-45-generic", 0, 0, 0, 0, ""},
		{"db-01", "db-01", "linux", "192.168.1.11", 34, 71, 32 << 30, 1.28, 51, 1209600, "6.8.0-45-generic", 0, 0, 0, 0, ""},
		{"nas-01", "nas-01", "linux", "192.168.1.20", 12, 55, 8 << 30, 0.21, 0, 2678400, "5.10.55", 0, 0, 0, 0, ""},
		{"router-01", "router-01", "ikuai", "192.168.1.1", 9, 38, 2 << 30, 0.05, 0, 864000, "", 14, 268, 1_800_000, 12_400_000, "203.0.113.7"},
	}

	snap := collector.Snapshot{Ts: time.Now().Unix()}
	for _, h := range hosts {
		snap.Hosts = append(snap.Hosts, h.snapshot())
	}

	type ct struct {
		host, name, image string
		running           bool
		cpu, memMB        float64
	}
	cts := []ct{
		{"web-01", "traefik", "traefik:v3", true, 1.2, 180},
		{"web-01", "nginx", "nginx:1.27", true, 0.6, 42},
		{"web-01", "whoami", "traefik/whoami", true, 0.1, 12},
		{"web-01", "watchtower", "containrrr/watchtower", true, 0.3, 28},
		{"web-01", "uptime-kuma", "louislam/uptime-kuma:1", false, 0, 0},
		{"db-01", "postgres", "postgres:16", true, 4.8, 1900},
		{"db-01", "redis", "redis:7", true, 1.1, 260},
		{"db-01", "minio", "minio/minio", true, 2.4, 620},
		{"db-01", "pgadmin", "dpage/pgadmin4", false, 0, 0},
		{"nas-01", "jellyfin", "jellyfin/jellyfin", true, 6.2, 780},
		{"nas-01", "immich", "ghcr.io/immich-app/immich-server", true, 3.1, 540},
		{"nas-01", "gitea", "gitea/gitea:1", true, 1.4, 310},
		{"nas-01", "grafana", "grafana/grafana", true, 1.9, 240},
		{"nas-01", "prometheus", "prom/prometheus", true, 2.7, 480},
		{"nas-01", "frigate", "ghcr.io/blakeblackshear/frigate", true, 8.4, 1200},
		{"nas-01", "sonarr", "linuxserver/sonarr", false, 0, 0},
		{"nas-01", "radarr", "linuxserver/radarr", false, 0, 0},
		{"nas-01", "photoprism", "photoprism/photoprism", false, 0, 0},
	}
	for _, c := range cts {
		state, status := "exited", "Exited (0) 2 days ago"
		if c.running {
			state, status = "running", "Up 3 hours (healthy)"
		}
		snap.Containers = append(snap.Containers, collector.Container{
			Host: c.host, Name: c.name, Image: c.image, State: state, Status: status,
			CPU: c.cpu, Mem: uint64(c.memMB) << 20, MemLimit: 2 << 30, HasStats: c.running,
		})
	}

	snap.PVE = collector.PVEView{
		Nodes: []collector.PVENode{
			{Name: "pve-01", Online: true, CPU: 14.2, MemUsed: 21 << 30, MemTotal: 32 << 30, Uptime: 2400000},
			{Name: "pve-02", Online: true, CPU: 8.6, MemUsed: 12 << 30, MemTotal: 32 << 30, Uptime: 900000},
		},
		Guests: []collector.PVEGuest{
			{Node: "pve-01", ID: 100, Name: "vm-docker", Kind: "qemu", Status: "running", CPU: 22.4, Mem: 6 << 30, MaxMem: 8 << 30, Disk: 80 << 30, MaxDisk: 200 << 30},
			{Node: "pve-01", ID: 101, Name: "vm-k8s", Kind: "qemu", Status: "running", CPU: 31.1, Mem: 12 << 30, MaxMem: 16 << 30, Disk: 120 << 30, MaxDisk: 300 << 30},
			{Node: "pve-01", ID: 102, Name: "vm-stopped", Kind: "qemu", Status: "stopped", MaxMem: 4 << 30, MaxDisk: 100 << 30},
			{Node: "pve-02", ID: 200, Name: "ct-homepage", Kind: "lxc", Status: "running", CPU: 1.2, Mem: 512 << 20, MaxMem: 1 << 30, Disk: 4 << 30, MaxDisk: 16 << 30},
			{Node: "pve-02", ID: 201, Name: "ct-backup", Kind: "lxc", Status: "stopped", MaxMem: 2 << 30, MaxDisk: 32 << 30},
		},
	}

	cert := 84
	snap.Services = []collector.ServiceStatus{
		{Name: "Homepage", URL: "https://home.example.com", OK: true, Code: 200, LatencyMS: 24, CertDays: &cert},
		{Name: "Grafana", URL: "https://grafana.example.com", OK: true, Code: 200, LatencyMS: 41, CertDays: &cert},
		{Name: "Git", URL: "https://git.example.com", OK: true, Code: 200, LatencyMS: 13, CertDays: &cert},
		{Name: "NAS", URL: "http://192.168.1.20/", OK: true, Code: 200, LatencyMS: 7},
		{Name: "PVE", URL: "https://192.168.1.4:8006/", OK: true, Code: 200, LatencyMS: 19, CertDays: &cert},
		{Name: "Status", URL: "https://status.example.com", OK: false, Code: 0, LatencyMS: 0, Error: "timeout"},
	}

	snap.Alerts = []collector.Alert{{Level: "error", Text: "Status 探活超时"}}

	for i := range snap.Hosts {
		hist := make([]float64, 0, 36)
		base := snap.Hosts[i].CPU
		for k := 0; k < 36; k++ {
			hist = append(hist, clamp(base+rand.NormFloat64()*3, 0, 100))
		}
		snap.Hosts[i].CPUHist = hist
	}

	s.snap = snap
}

// Start jitters the metrics so the panel feels alive.
func (s *Source) Start(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick()
		}
	}
}

func (s *Source) tick() {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := s.snap
	snap.Ts = time.Now().Unix()
	for i := range snap.Hosts {
		h := &snap.Hosts[i]
		h.CPU = clamp(h.CPU+rand.NormFloat64()*4, 0, 100)
		h.Load1 = math.Max(0, h.Load1+rand.NormFloat64()*0.08)
		if h.Kind != "ikuai" {
			h.NetRx = uint64(math.Max(0, float64(h.NetRx)+(rand.Float64()-0.5)*3000))
			h.NetTx = uint64(math.Max(0, float64(h.NetTx)+(rand.Float64()-0.5)*1200))
		}
		hist := append(h.CPUHist, h.CPU)
		if len(hist) > 36 {
			hist = hist[len(hist)-36:]
		}
		h.CPUHist = hist
	}
	for i := range snap.Containers {
		c := &snap.Containers[i]
		if c.State == "running" {
			c.CPU = clamp(c.CPU+rand.NormFloat64()*0.4, 0.01, 90)
		}
	}
	for i := range snap.PVE.Nodes {
		n := &snap.PVE.Nodes[i]
		n.CPU = clamp(n.CPU+rand.NormFloat64()*2, 0, 100)
	}
	s.snap = snap
}

// Snapshot returns the latest synthetic snapshot.
func (s *Source) Snapshot() collector.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

// History returns synthetic samples for the trend page.
func (s *Source) History(host string, from, to int64, limit int) ([]store.Sample, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	all := s.hist[host]
	out := make([]store.Sample, 0, len(all))
	for _, sample := range all {
		if sample.Ts >= from && sample.Ts <= to {
			out = append(out, sample)
		}
	}
	if limit > 0 && len(out) > limit {
		step := len(out) / limit
		if step < 1 {
			step = 1
		}
		down := make([]store.Sample, 0, limit)
		for i := 0; i < len(out); i += step {
			down = append(down, out[i])
		}
		out = down
	}
	return out, nil
}

// seedHistory fabricates six hours of one-minute samples per host.
func (s *Source) seedHistory() {
	now := time.Now().Unix()
	for _, h := range s.snap.Hosts {
		base := h.CPU
		memTotal := h.MemTotal
		memBase := 0.0
		if memTotal > 0 {
			memBase = float64(h.MemUsed) / float64(memTotal) * 100
		}
		samples := make([]store.Sample, 0, histLen)
		for i := histLen; i > 0; i-- {
			ts := now - int64(i*60)
			phase := float64(i) / 22
			cpu := clamp(base+8*math.Sin(phase)+rand.NormFloat64()*3, 0, 100)
			mem := clamp(memBase+3*math.Sin(phase/3)+rand.NormFloat64()*0.6, 0, 100)
			samples = append(samples, store.Sample{
				Ts: ts, CPU: cpu,
				MemUsed: uint64(float64(memTotal) * mem / 100), MemTotal: memTotal,
				DiskUsed: memTotal / 2, DiskTotal: memTotal,
				Load1: math.Max(0, h.Load1+rand.NormFloat64()*0.1),
			})
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i].Ts < samples[j].Ts })
		s.hist[h.ID] = samples
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

var _ = strconv.Itoa
