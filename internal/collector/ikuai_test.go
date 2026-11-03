package collector

import (
	"encoding/json"
	"testing"
)

func TestIKuaiCPUAvg(t *testing.T) {
	cases := []struct {
		in   []string
		want float64
	}{
		{nil, 0},
		{[]string{"1.25%"}, 1.25},
		{[]string{"1.25%", "2.75%", "0.00%", "4.00%"}, 2},
		{[]string{"bad", "3.5%"}, 3.5},
		{[]string{"%", ""}, 0},
	}
	for _, c := range cases {
		if got := ikuaiCPUAvg(c.in); got != c.want {
			t.Errorf("ikuaiCPUAvg(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestIKuaiEnvelopeParsing(t *testing.T) {
	raw := `{
      "code": 0, "message": "Success",
      "results": {"sysstat": {
        "cpu": ["1.25%","2.75%","1.00%","3.00%"],
        "hostname": "iKuai", "ip_addr": "192.168.1.1", "uptime": 537729,
        "memory": {"total": 1902492, "available": 1276196, "free": 900000,
                   "cached": 100000, "buffers": 1000, "used": "32%"},
        "online_user": {"count": 16, "count_wired": 9, "count_wireless": 7},
        "stream": {"upload": 141438, "download": 154871,
                   "total_up": 1, "total_down": 2, "connect_num": 307},
        "verinfo": {"version": "4.0.313"}
      }}
    }`
	var env ikuaiEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Code != 0 || env.Results.Sysstat == nil {
		t.Fatalf("unexpected envelope: %+v", env)
	}
	s := env.Results.Sysstat
	if got := ikuaiCPUAvg(s.CPU); got != 2 {
		t.Errorf("cpu = %v, want 2", got)
	}
	if s.Memory.Total != 1902492 {
		t.Errorf("mem total = %d", s.Memory.Total)
	}
	if s.OnlineUser.Count != 16 || s.Stream.ConnectNum != 307 {
		t.Errorf("clients/conns wrong: %+v %+v", s.OnlineUser, s.Stream)
	}
	if s.VerInfo.Version != "4.0.313" {
		t.Errorf("version = %q", s.VerInfo.Version)
	}
}

func TestIKuaiEnvelopeIfaceCheck(t *testing.T) {
	raw := `{"code":0,"results":{"iface_check":[
      {"interface":"wan1","ip_addr":"10.17.122.51","internet":"1","result":"success"}]}}`
	var env ikuaiEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(env.Results.IfaceCheck) != 1 {
		t.Fatalf("iface_check len = %d", len(env.Results.IfaceCheck))
	}
	if got := env.Results.IfaceCheck[0]; got.Result != "success" || got.IPAddr != "10.17.122.51" {
		t.Errorf("unexpected iface: %+v", got)
	}
}
