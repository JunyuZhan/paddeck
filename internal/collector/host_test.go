package collector

import "testing"

const sampleHostOutput = `@@STAT
cpu  100 0 100 800 0 0 0 0 0 0
@@MEM
MemTotal:        8000000 kB
MemAvailable:    6000000 kB
MemFree:         1000000 kB
Buffers:          500000 kB
Cached:          4000000 kB
SwapTotal:       2000000 kB
SwapFree:        1900000 kB
@@LOAD
0.42 0.31 0.20 1/234 5678
@@UPTIME
123456.78 234567.89
@@DISK
Filesystem 1024-blocks Used Available Capacity Mounted on
/dev/sda1 100000000 40000000 60000000 40% /
/dev/sdb1 200000000 10000000 190000000 5% /data
@@HOST
web-01 6.12.0-1
@@TEMP
42000
`

func TestParseHostOutput(t *testing.T) {
	m := parseHostOutput(sampleHostOutput, []string{"/"})
	if m.MemTotal != 8000000*1024 {
		t.Errorf("MemTotal = %d", m.MemTotal)
	}
	if m.MemUsed != 2000000*1024 {
		t.Errorf("MemUsed = %d", m.MemUsed)
	}
	if m.DiskUsed != 40000000*1024 || m.DiskTotal != 100000000*1024 {
		t.Errorf("disk = %d/%d", m.DiskUsed, m.DiskTotal)
	}
	if m.Load1 != 0.42 {
		t.Errorf("Load1 = %v", m.Load1)
	}
	if m.Uptime != 123456 {
		t.Errorf("Uptime = %d", m.Uptime)
	}
	if m.Hostname != "web-01" || m.Kernel != "6.12.0-1" {
		t.Errorf("host = %q %q", m.Hostname, m.Kernel)
	}
	if m.Temp != 42 {
		t.Errorf("Temp = %v", m.Temp)
	}
}

func TestParseDiskPrefersConfiguredMount(t *testing.T) {
	used, total := parseDisk(sections(sampleHostOutput)["DISK"], []string{"/data"})
	if used != 10000000*1024 || total != 200000000*1024 {
		t.Errorf("disk = %d/%d", used, total)
	}
}

func TestParseStatLine(t *testing.T) {
	s, ok := parseStatLine("cpu  100 0 100 800 0 0 0 0 0 0")
	if !ok {
		t.Fatal("not ok")
	}
	if s.total != 1000 || s.idle != 800 {
		t.Errorf("total=%d idle=%d", s.total, s.idle)
	}
}
