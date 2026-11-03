package collector

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MacCollector collects macOS metrics over SSH.
//
// macOS has no /proc, so the probe leans on top(1), vm_stat(1), sysctl(1)
// and df(1). top takes ~2s because it needs two samples to report CPU.
type MacCollector struct {
	ssh      *SSH
	mu       sync.Mutex
	previous map[string]time.Time
}

func NewMacCollector(ssh *SSH) *MacCollector {
	return &MacCollector{ssh: ssh, previous: map[string]time.Time{}}
}

const macScript = `export LC_ALL=C
echo "@@CPU"
top -l 2 -n 0 -s 1 2>/dev/null | grep -E '^CPU usage' | tail -1
echo "@@MEM"
vm_stat 2>/dev/null
echo "@@SYS"
sysctl -n hw.memsize 2>/dev/null
sysctl -n hw.pagesize 2>/dev/null
sysctl -n kern.boottime 2>/dev/null
sysctl -n vm.loadavg 2>/dev/null
echo "@@DISK"
df -k / 2>/dev/null
echo "@@HOST"
hostname 2>/dev/null
uname -r 2>/dev/null
exit 0
`

// Collect runs the probe and returns one sample.
func (c *MacCollector) Collect(ctx context.Context, key string) HostMetrics {
	now := time.Now().Unix()
	out, err := c.ssh.Run(ctx, macScript)
	if err != nil {
		return HostMetrics{Online: false, LastSeen: now, Err: err.Error()}
	}
	m := parseMacOutput(out)
	m.Online = true
	m.LastSeen = now
	return m
}

func parseMacOutput(out string) HostMetrics {
	s := sections(out)
	m := HostMetrics{}

	if line := strings.TrimSpace(s["CPU"]); line != "" {
		// CPU usage: 19.78% user, 14.99% sys, 65.21% idle
		if i := strings.Index(line, "idle"); i > 0 {
			part := line[:i]
			if j := strings.LastIndex(part, ","); j >= 0 {
				part = part[j+1:]
			}
			part = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(part), "%"))
			// part is like "65.21% idle" -> take the leading number
			fields := strings.Fields(part)
			if len(fields) > 0 {
				if idle, err := strconv.ParseFloat(strings.TrimSuffix(fields[0], "%"), 64); err == nil {
					m.CPU = round2(100 - idle)
				}
			}
		}
	}

	sys := strings.Split(strings.TrimSpace(s["SYS"]), "\n")
	var memTotal, pageSize uint64
	var bootSec int64
	for i, v := range sys {
		v = strings.TrimSpace(v)
		switch i {
		case 0:
			memTotal, _ = strconv.ParseUint(v, 10, 64)
		case 1:
			pageSize, _ = strconv.ParseUint(v, 10, 64)
		case 2:
			// { sec = 1793174631, usec = 816088 } ...
			if k := strings.Index(v, "sec = "); k >= 0 {
				rest := strings.TrimSpace(v[k+len("sec = "):])
				if c := strings.IndexAny(rest, ",}"); c > 0 {
					bootSec, _ = strconv.ParseInt(strings.TrimSpace(rest[:c]), 10, 64)
				}
			}
		case 3:
			// { 2.82 3.19 3.31 }
			nums := strings.Fields(strings.Trim(v, "{} "))
			if len(nums) >= 3 {
				m.Load1, _ = strconv.ParseFloat(nums[0], 64)
				m.Load5, _ = strconv.ParseFloat(nums[1], 64)
				m.Load15, _ = strconv.ParseFloat(nums[2], 64)
			}
		}
	}
	if pageSize == 0 {
		pageSize = 4096
	}
	if memTotal > 0 {
		m.MemTotal = memTotal
		availPages := macAvailablePages(s["MEM"])
		avail := availPages * pageSize
		if m.MemTotal > avail {
			m.MemUsed = m.MemTotal - avail
		}
	}
	if bootSec > 0 {
		m.Uptime = uint64(time.Now().Unix() - bootSec)
	}

	if rows := strings.Split(s["DISK"], "\n"); len(rows) > 1 {
		if f := strings.Fields(rows[1]); len(f) >= 3 {
			if total, err := strconv.ParseUint(f[1], 10, 64); err == nil {
				if used, err2 := strconv.ParseUint(f[2], 10, 64); err2 == nil {
					m.DiskTotal, m.DiskUsed = total*1024, used*1024
				}
			}
		}
	}

	if h := strings.Fields(s["HOST"]); len(h) >= 2 {
		m.Hostname = h[0]
		m.Kernel = h[1]
	}
	return m
}

// macAvailablePages sums the page counts macOS would hand back to apps.
func macAvailablePages(vmstat string) uint64 {
	var free, inactive, speculative uint64
	for _, line := range strings.Split(vmstat, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		key := strings.TrimSuffix(strings.Join(f[:len(f)-1], " "), ":")
		v, err := strconv.ParseUint(strings.TrimSuffix(f[len(f)-1], "."), 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "Pages free":
			free = v
		case "Pages inactive":
			inactive = v
		case "Pages speculative":
			speculative = v
		}
	}
	return free + inactive + speculative
}
