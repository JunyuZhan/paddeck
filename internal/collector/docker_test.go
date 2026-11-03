package collector

import "testing"

func TestParseMemUsage(t *testing.T) {
	used, limit := parseMemUsage("1.5GiB / 7.8GiB")
	if used != uint64(1.5*1024*1024*1024) {
		t.Errorf("used = %d", used)
	}
	if limit != 8375186227 {
		t.Errorf("limit = %d", limit)
	}
}

func TestParseHumanBytes(t *testing.T) {
	cases := map[string]uint64{
		"0B":    0,
		"512B":  512,
		"1KiB":  1024,
		"2MiB":  2 * 1024 * 1024,
		"1.5GB": 1500000000,
	}
	for in, want := range cases {
		if got := parseHumanBytes(in); got != want {
			t.Errorf("%s = %d, want %d", in, got, want)
		}
	}
}

func TestParsePercent(t *testing.T) {
	if got := parsePercent("12.34%"); got != 12.34 {
		t.Errorf("got %v", got)
	}
}
