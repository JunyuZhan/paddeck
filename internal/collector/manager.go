package collector

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/JunyuZhan/paddeck/internal/config"
	"github.com/JunyuZhan/paddeck/internal/store"
)

// HostView is a host plus registry metadata.
type HostView struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Address     string    `json:"address"`
	Description string    `json:"description,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
	CPUHist     []float64 `json:"cpu_hist,omitempty"`
	HostMetrics
}

// PVEView bundles nodes and guests.
type PVEView struct {
	Nodes  []PVENode  `json:"nodes"`
	Guests []PVEGuest `json:"guests"`
}

// Alert is a derived problem shown on the panel.
type Alert struct {
	Level string `json:"level"` // error | warn
	Text  string `json:"text"`
}

// Snapshot is the complete state served to the frontend.
type Snapshot struct {
	Ts         int64           `json:"ts"`
	Hosts      []HostView      `json:"hosts"`
	Containers []Container     `json:"containers"`
	PVE        PVEView         `json:"pve"`
	Services   []ServiceStatus `json:"services"`
	Alerts     []Alert         `json:"alerts"`
}

// metricsCollector is the minimal surface the manager needs from a collector.
type metricsCollector interface {
	Collect(ctx context.Context, key string) HostMetrics
}

// cpuHistLen is how many samples the in-memory CPU sparkline keeps.
const cpuHistLen = 36

// Manager schedules collectors and keeps the latest snapshot.
type Manager struct {
	cfg   *config.Config
	store *store.Store

	hostCollectors   map[string]metricsCollector
	dockerCollectors map[string]*DockerCollector
	pveCollectors    map[string]*PVECollector
	prober           *Prober

	mu      sync.RWMutex
	snap    Snapshot
	cpuHist map[string][]float64

	lastDocker time.Time
	lastPVE    time.Time
	lastWrite  map[string]int64
	startedAt  time.Time
}

func NewManager(cfg *config.Config, st *store.Store) *Manager {
	m := &Manager{
		cfg:              cfg,
		store:            st,
		hostCollectors:   map[string]metricsCollector{},
		dockerCollectors: map[string]*DockerCollector{},
		pveCollectors:    map[string]*PVECollector{},
		prober:           NewProber(cfg.Collector.HTTPTimeout.Std()),
		cpuHist:          map[string][]float64{},
		lastWrite:        map[string]int64{},
		startedAt:        time.Now(),
	}
	for _, h := range cfg.Hosts {
		if h.IsIKuai() {
			m.hostCollectors[h.ID] = NewIKuaiCollector(h.APIURL, h.Username, h.Password, cfg.Collector.SSHTimeout.Std())
			continue
		}
		ssh := &SSH{Addr: h.Address, User: h.User, Port: h.Port, KeyPath: h.SSHKey, Timeout: cfg.Collector.SSHTimeout.Std()}
		if h.Type == "macos" {
			m.hostCollectors[h.ID] = NewMacCollector(ssh)
			continue
		}
		m.hostCollectors[h.ID] = NewHostCollector(ssh, h.DiskMounts)
		if h.Docker {
			m.dockerCollectors[h.ID] = NewDockerCollector(ssh, h.DockerPath, h.DockerSudo)
		}
	}
	for _, p := range cfg.PVE {
		ssh := &SSH{Addr: p.Address, User: p.User, Port: p.Port, KeyPath: p.SSHKey, Timeout: cfg.Collector.SSHTimeout.Std()}
		m.pveCollectors[p.ID] = NewPVECollector(ssh, p.Name)
	}
	return m
}

// Start runs the collection loop until ctx is cancelled.
func (m *Manager) Start(ctx context.Context) {
	m.collect(ctx)
	ticker := time.NewTicker(m.cfg.Collector.Interval.Std())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collect(ctx)
		}
	}
}

// Snapshot returns the latest snapshot.
func (m *Manager) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := m.snap
	if snap.Hosts == nil {
		snap.Hosts = []HostView{}
	}
	if snap.Containers == nil {
		snap.Containers = []Container{}
	}
	if snap.Services == nil {
		snap.Services = []ServiceStatus{}
	}
	if snap.Alerts == nil {
		snap.Alerts = []Alert{}
	}
	if snap.PVE.Nodes == nil {
		snap.PVE.Nodes = []PVENode{}
	}
	if snap.PVE.Guests == nil {
		snap.PVE.Guests = []PVEGuest{}
	}
	return snap
}

func (m *Manager) collect(ctx context.Context) {
	now := time.Now()
	snap := Snapshot{Ts: now.Unix()}

	hosts := m.collectHosts(ctx)
	m.trackCPU(hosts)
	snap.Hosts = hosts

	if now.Sub(m.lastDocker) >= m.cfg.Collector.DockerInterval.Std() {
		m.lastDocker = now
	}
	snap.Containers = m.collectContainers(ctx, hosts)

	if now.Sub(m.lastPVE) >= m.cfg.Collector.DockerInterval.Std() {
		m.lastPVE = now
	}
	snap.PVE = m.collectPVE(ctx)

	snap.Services = m.collectServices(ctx)
	snap.Alerts = buildAlerts(hosts, snap.Containers, snap.Services)

	m.mu.Lock()
	m.snap = snap
	m.mu.Unlock()

	m.writeHistory(now, hosts)
}

func (m *Manager) collectHosts(ctx context.Context) []HostView {
	type result struct {
		idx int
		m   HostMetrics
	}
	results := make(chan result, len(m.cfg.Hosts))
	var wg sync.WaitGroup

	for i, h := range m.cfg.Hosts {
		wg.Add(1)
		go func(i int, h config.Host) {
			defer wg.Done()
			c := m.hostCollectors[h.ID]
			cctx, cancel := context.WithTimeout(ctx, m.cfg.Collector.SSHTimeout.Std())
			defer cancel()
			results <- result{idx: i, m: c.Collect(cctx, h.ID)}
		}(i, h)
	}
	wg.Wait()
	close(results)

	out := make([]HostView, len(m.cfg.Hosts))
	for i, h := range m.cfg.Hosts {
		out[i] = HostView{ID: h.ID, Name: h.Name, Kind: h.Type, Address: h.Address, Description: h.Description, Labels: h.Labels}
	}
	for r := range results {
		out[r.idx].HostMetrics = r.m
	}

	// Keep last good values for offline hosts.
	m.mu.RLock()
	prev := m.snap.Hosts
	m.mu.RUnlock()
	for i := range out {
		if out[i].Online {
			continue
		}
		for _, p := range prev {
			if p.ID == out[i].ID && p.Online {
				last := p
				last.Online = false
				last.Err = out[i].Err
				last.LastSeen = p.LastSeen
				out[i] = HostView{
					ID: out[i].ID, Name: out[i].Name, Kind: out[i].Kind, Address: out[i].Address,
					Description: out[i].Description, Labels: out[i].Labels,
					HostMetrics: last.HostMetrics,
				}
				break
			}
		}
	}
	return out
}

// trackCPU appends the current CPU sample to each online host's in-memory
// ring buffer and attaches the series to the view for the tile sparkline.
func (m *Manager) trackCPU(hosts []HostView) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range hosts {
		id := hosts[i].ID
		if !hosts[i].Online {
			continue
		}
		s := append(m.cpuHist[id], hosts[i].CPU)
		if len(s) > cpuHistLen {
			s = s[len(s)-cpuHistLen:]
		}
		m.cpuHist[id] = s
		hist := make([]float64, len(s))
		copy(hist, s)
		hosts[i].CPUHist = hist
	}
}

func (m *Manager) collectContainers(ctx context.Context, hosts []HostView) []Container {
	online := map[string]bool{}
	for _, h := range hosts {
		online[h.ID] = h.Online
	}
	var mu sync.Mutex
	var all []Container
	var wg sync.WaitGroup
	for id, dc := range m.dockerCollectors {
		if !online[id] {
			continue
		}
		wg.Add(1)
		go func(id string, dc *DockerCollector) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			list, err := dc.List(cctx)
			if err != nil {
				log.Printf("docker list %s: %v", id, err)
				return
			}
			stats, _ := dc.Stats(cctx)
			for i := range list {
				list[i].Host = id
				if s, ok := stats[list[i].Name]; ok {
					list[i].CPU = s.CPU
					list[i].Mem = s.Mem
					list[i].MemLimit = s.MemLimit
					list[i].Restarts = s.Restarts
					list[i].HasStats = s.HasStats
				}
			}
			mu.Lock()
			all = append(all, list...)
			mu.Unlock()
		}(id, dc)
	}
	wg.Wait()
	sort.Slice(all, func(i, j int) bool {
		if all[i].Host != all[j].Host {
			return all[i].Host < all[j].Host
		}
		return all[i].Name < all[j].Name
	})
	return all
}

func (m *Manager) collectPVE(ctx context.Context) PVEView {
	var view PVEView
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, p := range m.cfg.PVE {
		pc := m.pveCollectors[p.ID]
		if pc == nil {
			continue
		}
		wg.Add(1)
		go func(id string, pc *PVECollector) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			nodes, guests, err := pc.Collect(cctx)
			if err != nil {
				log.Printf("pve %s: %v", id, err)
				return
			}
			mu.Lock()
			view.Nodes = append(view.Nodes, nodes...)
			view.Guests = append(view.Guests, guests...)
			mu.Unlock()
		}(p.ID, pc)
	}
	wg.Wait()
	sort.Slice(view.Nodes, func(i, j int) bool { return view.Nodes[i].Name < view.Nodes[j].Name })
	sort.Slice(view.Guests, func(i, j int) bool {
		if view.Guests[i].Node != view.Guests[j].Node {
			return view.Guests[i].Node < view.Guests[j].Node
		}
		return view.Guests[i].ID < view.Guests[j].ID
	})
	return view
}

func (m *Manager) collectServices(ctx context.Context) []ServiceStatus {
	type result struct {
		idx int
		s   ServiceStatus
	}
	results := make(chan result, len(m.cfg.Services))
	var wg sync.WaitGroup
	for i, svc := range m.cfg.Services {
		wg.Add(1)
		go func(i int, svc config.Service) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, m.cfg.Collector.HTTPTimeout.Std())
			defer cancel()
			results <- result{idx: i, s: m.prober.Probe(cctx, svc.Name, svc.URL)}
		}(i, svc)
	}
	wg.Wait()
	close(results)

	out := make([]ServiceStatus, len(m.cfg.Services))
	for r := range results {
		out[r.idx] = r.s
	}
	return out
}

func (m *Manager) writeHistory(now time.Time, hosts []HostView) {
	if m.store == nil {
		return
	}
	if now.Sub(m.startedAt) < m.cfg.Collector.HistoryInterval.Std() {
		return
	}
	for _, h := range hosts {
		if !h.Online {
			continue
		}
		if now.Unix()-m.lastWrite[h.ID] < int64(m.cfg.Collector.HistoryInterval.Std().Seconds()) {
			continue
		}
		m.lastWrite[h.ID] = now.Unix()
		if err := m.store.Insert(h.ID, now.Unix(), h.CPU, h.MemUsed, h.MemTotal, h.DiskUsed, h.DiskTotal, h.Load1); err != nil {
			log.Printf("history %s: %v", h.ID, err)
		}
	}
}

func buildAlerts(hosts []HostView, containers []Container, services []ServiceStatus) []Alert {
	alerts := []Alert{}
	now := time.Now().Unix()
	for _, h := range hosts {
		if !h.Online {
			text := fmt.Sprintf("%s 离线", h.Name)
			if h.LastSeen > 0 {
				text = fmt.Sprintf("%s 离线（最后在线 %ds 前）", h.Name, now-h.LastSeen)
			}
			alerts = append(alerts, Alert{Level: "error", Text: text})
			continue
		}
		if h.CPU >= 90 {
			alerts = append(alerts, Alert{Level: "warn", Text: fmt.Sprintf("%s CPU %.0f%%", h.Name, h.CPU)})
		}
		if h.MemTotal > 0 && float64(h.MemUsed)/float64(h.MemTotal) >= 0.9 {
			alerts = append(alerts, Alert{Level: "warn", Text: fmt.Sprintf("%s 内存 %.0f%%", h.Name, memPct(h.MemUsed, h.MemTotal))})
		}
		if h.DiskTotal > 0 && float64(h.DiskUsed)/float64(h.DiskTotal) >= 0.9 {
			alerts = append(alerts, Alert{Level: "warn", Text: fmt.Sprintf("%s 磁盘 %.0f%%", h.Name, diskPct(h.DiskUsed, h.DiskTotal))})
		}
	}
	for _, s := range services {
		if !s.OK {
			alerts = append(alerts, Alert{Level: "error", Text: fmt.Sprintf("%s 不可用", s.Name)})
		}
	}
	if len(alerts) > 20 {
		alerts = alerts[:20]
	}
	return alerts
}

func memPct(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}

func diskPct(used, total uint64) float64 { return memPct(used, total) }
