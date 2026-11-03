package collector

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HostMetrics is one sample of a host. The first block is shared by every
// collector; the fields after DiskTotal are extra signals some devices report.
type HostMetrics struct {
	CPU       float64 `json:"cpu"`
	MemUsed   uint64  `json:"mem_used"`
	MemTotal  uint64  `json:"mem_total"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskTotal uint64  `json:"disk_total"`
	Load1     float64 `json:"load1"`
	Load5     float64 `json:"load5"`
	Load15    float64 `json:"load15"`
	Uptime    uint64  `json:"uptime"`
	Temp      float64 `json:"temp"`
	Kernel    string  `json:"kernel"`
	Hostname  string  `json:"hostname"`
	Online    bool    `json:"online"`
	LastSeen  int64   `json:"last_seen"`
	Err       string  `json:"err,omitempty"`

	// Throughput in bytes/s (Linux hosts).
	NetRx uint64 `json:"net_rx,omitempty"`
	NetTx uint64 `json:"net_tx,omitempty"`

	// Router extras (iKuai). Zero on Linux hosts.
	Clients   int    `json:"clients,omitempty"`
	Conns     int    `json:"conns,omitempty"`
	WANUp     uint64 `json:"wan_up,omitempty"`
	WANDown   uint64 `json:"wan_down,omitempty"`
	WANIP     string `json:"wan_ip,omitempty"`
	WANStatus string `json:"wan_status,omitempty"`
}

const hostScript = `export LC_ALL=C
echo "@@STAT"
head -1 /proc/stat 2>/dev/null
sleep 0.3
head -1 /proc/stat 2>/dev/null
echo "@@MEM"
grep -E '^(MemTotal|MemAvailable|MemFree|Buffers|Cached|SwapTotal|SwapFree):' /proc/meminfo 2>/dev/null
echo "@@LOAD"
cat /proc/loadavg 2>/dev/null
echo "@@UPTIME"
cat /proc/uptime 2>/dev/null
echo "@@DISK"
df -P -x tmpfs -x devtmpfs -x squashfs 2>/dev/null || df -P 2>/dev/null
echo "@@HOST"
cat /proc/sys/kernel/hostname 2>/dev/null
uname -r 2>/dev/null
echo "@@TEMP"
cat /sys/class/thermal/thermal_zone*/temp 2>/dev/null | sort -n | tail -1
echo "@@NET"
cat /proc/net/dev 2>/dev/null
exit 0
`

type cpuSample struct {
	idle  uint64
	total uint64
}

// netSample remembers the counters of the previous poll so throughput can be
// derived without asking the kernel for a rate.
type netSample struct {
	at time.Time
	rx uint64
	tx uint64
}

// HostCollector collects Linux metrics over SSH and computes CPU deltas.
type HostCollector struct {
	ssh         *SSH
	mounts      []string
	mu          sync.Mutex
	previous    map[string]cpuSample
	previousNet map[string]netSample
}

func NewHostCollector(ssh *SSH, mounts []string) *HostCollector {
	return &HostCollector{
		ssh:         ssh,
		mounts:      mounts,
		previous:    map[string]cpuSample{},
		previousNet: map[string]netSample{},
	}
}

// Collect runs the probe and computes CPU usage for the given key.
func (c *HostCollector) Collect(ctx context.Context, key string) HostMetrics {
	now := time.Now().Unix()
	out, err := c.ssh.Run(ctx, hostScript)
	if err != nil {
		return HostMetrics{Online: false, LastSeen: now, Err: err.Error()}
	}
	m := parseHostOutput(out, c.mounts)
	m.Online = true
	m.LastSeen = now

	c.mu.Lock()
	lines := strings.Split(strings.TrimSpace(sections(out)["STAT"]), "\n")
	if len(lines) >= 1 {
		first, okFirst := parseStatLine(lines[0])
		last, okLast := parseStatLine(lines[len(lines)-1])
		if okFirst && okLast && last.total > first.total {
			// First sample in this process: use the two in-script reads.
			if _, exists := c.previous[key]; !exists {
				m.CPU = cpuPercent(first, last)
			}
		}
		if okLast {
			if prev, exists := c.previous[key]; exists && last.total > prev.total {
				m.CPU = cpuPercent(prev, last)
			}
			c.previous[key] = last
		}
	}

	rx, tx := parseNet(sections(out)["NET"])
	if prev, exists := c.previousNet[key]; exists {
		dt := time.Since(prev.at).Seconds()
		if dt > 0.5 && rx >= prev.rx && tx >= prev.tx {
			m.NetRx = uint64(float64(rx-prev.rx) / dt)
			m.NetTx = uint64(float64(tx-prev.tx) / dt)
		}
	}
	c.previousNet[key] = netSample{at: time.Now(), rx: rx, tx: tx}
	c.mu.Unlock()

	return m
}

// parseNet sums rx/tx byte counters across physical interfaces.
func parseNet(block string) (rx, tx uint64) {
	for _, line := range strings.Split(block, "\n") {
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		name := strings.TrimSpace(line[:i])
		if name == "" || name == "lo" || name == "Inter-|" {
			continue
		}
		f := strings.Fields(line[i+1:])
		if len(f) < 16 {
			continue
		}
		r, err1 := strconv.ParseUint(f[0], 10, 64)
		t, err2 := strconv.ParseUint(f[8], 10, 64)
		if err1 == nil && err2 == nil {
			rx += r
			tx += t
		}
	}
	return rx, tx
}

func cpuPercent(prev, cur cpuSample) float64 {
	dTotal := cur.total - prev.total
	if dTotal == 0 {
		return 0
	}
	dIdle := cur.idle - prev.idle
	return round2(float64(dTotal-dIdle) / float64(dTotal) * 100)
}

func sections(out string) map[string]string {
	result := map[string]string{}
	var current string
	var b strings.Builder
	flush := func() {
		if current != "" {
			result[current] = strings.TrimSpace(b.String())
		}
		b.Reset()
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "@@") {
			flush()
			current = strings.TrimPrefix(line, "@@")
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	flush()
	return result
}

func parseStatLine(line string) (cpuSample, bool) {
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, false
	}
	vals := make([]uint64, 0, len(fields)-1)
	for _, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuSample{}, false
		}
		vals = append(vals, v)
	}
	var total uint64
	for _, v := range vals {
		total += v
	}
	idle := vals[3]
	if len(vals) > 4 {
		idle += vals[4] // iowait counts as idle
	}
	return cpuSample{idle: idle, total: total}, true
}

func parseHostOutput(out string, mounts []string) HostMetrics {
	s := sections(out)
	m := HostMetrics{}

	if f := strings.Fields(s["LOAD"]); len(f) >= 3 {
		m.Load1, _ = strconv.ParseFloat(f[0], 64)
		m.Load5, _ = strconv.ParseFloat(f[1], 64)
		m.Load15, _ = strconv.ParseFloat(f[2], 64)
	}
	if f := strings.Fields(s["UPTIME"]); len(f) >= 1 {
		if sec, err := strconv.ParseFloat(f[0], 64); err == nil {
			m.Uptime = uint64(sec)
		}
	}
	if h := strings.Fields(s["HOST"]); len(h) >= 2 {
		m.Hostname = h[0]
		m.Kernel = h[1]
	}

	mem := map[string]uint64{}
	for _, line := range strings.Split(s["MEM"], "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		if v, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
			mem[key] = v * 1024
		}
	}
	m.MemTotal = mem["MemTotal"]
	avail := mem["MemAvailable"]
	if avail == 0 {
		avail = mem["MemFree"] + mem["Buffers"] + mem["Cached"]
	}
	if m.MemTotal > avail {
		m.MemUsed = m.MemTotal - avail
	}

	used, total := parseDisk(s["DISK"], mounts)
	m.DiskUsed, m.DiskTotal = used, total

	if t, err := strconv.ParseFloat(strings.TrimSpace(s["TEMP"]), 64); err == nil {
		if t > 1000 {
			t = t / 1000
		}
		m.Temp = round2(t)
	}
	return m
}

func parseDisk(disk string, mounts []string) (used, total uint64) {
	lines := strings.Split(disk, "\n")
	if len(lines) < 2 {
		return 0, 0
	}
	type row struct {
		used  uint64
		total uint64
		mount string
	}
	var rows []row
	for _, line := range lines[1:] {
		f := strings.Fields(line)
		if len(f) < 6 {
			continue
		}
		blocks, err1 := strconv.ParseUint(f[1], 10, 64)
		u, err2 := strconv.ParseUint(f[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		rows = append(rows, row{used: u * 1024, total: blocks * 1024, mount: f[len(f)-1]})
	}
	want := mounts
	if len(want) == 0 {
		want = []string{"/"}
	}
	for _, mnt := range want {
		for _, r := range rows {
			if r.mount == mnt {
				return r.used, r.total
			}
		}
	}
	// fall back to the first row
	if len(rows) > 0 {
		return rows[0].used, rows[0].total
	}
	return 0, 0
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
