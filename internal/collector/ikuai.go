package collector

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IKuaiCollector reads router metrics from the iKuai web API.
//
// The API lives on https://<addr>:444 and uses a self-signed certificate.
// Login: POST /Action/login with the md5 of the password; the device answers
// with a "sess_key" cookie which is replayed on every /Action/call request.
type IKuaiCollector struct {
	baseURL  string
	username string
	password string
	client   *http.Client

	mu         sync.Mutex
	sessKey    string
	lastCookie string

	wanMu   sync.Mutex
	wanAt   time.Time
	wanInfo string
	wanIP   string
}

type ikuaiSysstat struct {
	CPU      []string `json:"cpu"`
	Hostname string   `json:"hostname"`
	IPAddr   string   `json:"ip_addr"`
	Uptime   uint64   `json:"uptime"`
	Memory   struct {
		Total     uint64 `json:"total"`
		Available uint64 `json:"available"`
		Free      uint64 `json:"free"`
		Cached    uint64 `json:"cached"`
		Buffers   uint64 `json:"buffers"`
		Used      string `json:"used"`
	} `json:"memory"`
	OnlineUser struct {
		Count         int `json:"count"`
		CountWired    int `json:"count_wired"`
		CountWireless int `json:"count_wireless"`
	} `json:"online_user"`
	Stream struct {
		Upload     uint64 `json:"upload"`
		Download   uint64 `json:"download"`
		TotalUp    uint64 `json:"total_up"`
		TotalDown  uint64 `json:"total_down"`
		ConnectNum int    `json:"connect_num"`
	} `json:"stream"`
	VerInfo struct {
		Version string `json:"version"`
	} `json:"verinfo"`
}

type ikuaiEnvelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Results struct {
		Sysstat    *ikuaiSysstat `json:"sysstat"`
		IfaceCheck []struct {
			Interface string `json:"interface"`
			IPAddr    string `json:"ip_addr"`
			Internet  string `json:"internet"`
			Result    string `json:"result"`
			ErrMsg    string `json:"errmsg"`
		} `json:"iface_check"`
	} `json:"results"`
}

func NewIKuaiCollector(baseURL, username, password string, timeout time.Duration) *IKuaiCollector {
	if !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &IKuaiCollector{
		baseURL:  baseURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // router self-signed cert
			},
		},
	}
}

func (c *IKuaiCollector) Collect(ctx context.Context, key string) HostMetrics {
	now := time.Now().Unix()
	sys, err := c.sysstat(ctx)
	if err != nil {
		// Session may have expired: drop it and retry once.
		c.mu.Lock()
		c.sessKey = ""
		c.lastCookie = ""
		c.mu.Unlock()
		sys, err = c.sysstat(ctx)
	}
	if err != nil {
		return HostMetrics{Online: false, LastSeen: now, Err: err.Error()}
	}

	m := HostMetrics{Online: true, LastSeen: now}
	m.CPU = ikuaiCPUAvg(sys.CPU)
	m.MemTotal = sys.Memory.Total * 1024
	avail := sys.Memory.Available
	if avail == 0 {
		avail = sys.Memory.Free + sys.Memory.Buffers + sys.Memory.Cached
	}
	if m.MemTotal > avail*1024 {
		m.MemUsed = m.MemTotal - avail*1024
	}
	m.Uptime = sys.Uptime
	m.Hostname = sys.Hostname
	if m.Hostname == "" {
		m.Hostname = "iKuai"
	}
	m.Kernel = sys.VerInfo.Version
	m.Clients = sys.OnlineUser.Count
	m.WANUp = sys.Stream.Upload
	m.WANDown = sys.Stream.Download
	m.Conns = sys.Stream.ConnectNum

	wan, ip := c.wanStatus(ctx)
	m.WANStatus = wan
	if ip != "" {
		m.WANIP = ip
	}
	return m
}

// sysstat fetches the homepage sysstat block.
func (c *IKuaiCollector) sysstat(ctx context.Context) (*ikuaiSysstat, error) {
	var env ikuaiEnvelope
	payload := map[string]any{
		"func_name": "homepage",
		"action":    "show",
		"param":     map[string]any{"TYPE": "sysstat"},
	}
	if err := c.call(ctx, payload, &env); err != nil {
		return nil, err
	}
	if env.Results.Sysstat == nil {
		return nil, fmt.Errorf("ikuai: empty sysstat")
	}
	return env.Results.Sysstat, nil
}

// wanStatus returns the state of the first WAN interface. Results are cached
// for a minute: link state changes slowly and this keeps the poll cheap.
func (c *IKuaiCollector) wanStatus(ctx context.Context) (status, ip string) {
	c.wanMu.Lock()
	defer c.wanMu.Unlock()
	if time.Since(c.wanAt) < time.Minute && c.wanInfo != "" {
		return c.wanInfo, c.wanIP
	}
	var env ikuaiEnvelope
	payload := map[string]any{
		"func_name": "monitor_iface",
		"action":    "show",
		"param":     map[string]any{"TYPE": "iface_check"},
	}
	if err := c.call(ctx, payload, &env); err != nil || len(env.Results.IfaceCheck) == 0 {
		return c.wanInfo, c.wanIP
	}
	iface := env.Results.IfaceCheck[0]
	status = "down"
	if iface.Result == "success" {
		status = "up"
	}
	c.wanAt = time.Now()
	c.wanInfo = status
	c.wanIP = iface.IPAddr
	return status, iface.IPAddr
}

// call performs a /Action/call request, logging in when no session is held.
func (c *IKuaiCollector) call(ctx context.Context, payload any, out *ikuaiEnvelope) error {
	if err := c.ensureSession(ctx); err != nil {
		return err
	}
	body, err := c.post(ctx, "/Action/call", payload)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("ikuai: parse response: %w", err)
	}
	if out.Code != 0 {
		return fmt.Errorf("ikuai: %s (code %d)", out.Message, out.Code)
	}
	return nil
}

func (c *IKuaiCollector) ensureSession(ctx context.Context) error {
	c.mu.Lock()
	key := c.sessKey
	c.mu.Unlock()
	if key != "" {
		return nil
	}
	sum := md5.Sum([]byte(c.password))
	payload := map[string]any{
		"username":          c.username,
		"passwd":            hex.EncodeToString(sum[:]),
		"pass":              "",
		"remember_password": "",
	}
	c.mu.Lock()
	c.lastCookie = ""
	c.mu.Unlock()
	body, err := c.post(ctx, "/Action/login", payload)
	if err != nil {
		return err
	}
	var res struct {
		Code    int    `json:"code"`
		Result  int    `json:"Result"`
		Message string `json:"message"`
		SessKey string `json:"sess_key"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("ikuai: parse login: %w", err)
	}
	if res.Code != 0 || (res.Result != 0 && res.Result != 10000) {
		return fmt.Errorf("ikuai: login failed: %s", res.Message)
	}
	// The session key arrives as a cookie; fall back to the body field.
	c.mu.Lock()
	if c.lastCookie != "" {
		c.sessKey = c.lastCookie
	} else if res.SessKey != "" {
		c.sessKey = res.SessKey
	}
	key = c.sessKey
	c.mu.Unlock()
	if key == "" {
		return fmt.Errorf("ikuai: login returned no session")
	}
	return nil
}

func (c *IKuaiCollector) post(ctx context.Context, path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.mu.Lock()
	if c.sessKey != "" {
		req.Header.Set("Cookie", "sess_key="+c.sessKey)
	}
	c.mu.Unlock()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ikuai: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("ikuai: read body: %w", err)
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "sess_key" && ck.Value != "" {
			c.mu.Lock()
			c.lastCookie = ck.Value
			if c.sessKey == "" {
				c.sessKey = ck.Value
			}
			c.mu.Unlock()
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ikuai: http %d", resp.StatusCode)
	}
	return body, nil
}

// ikuaiCPUAvg averages the per-core percentage strings ("1.25%").
func ikuaiCPUAvg(values []string) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	var n int
	for _, raw := range values {
		v, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(raw), "%"), 64)
		if err != nil {
			continue
		}
		sum += v
		n++
	}
	if n == 0 {
		return 0
	}
	return round2(sum / float64(n))
}
