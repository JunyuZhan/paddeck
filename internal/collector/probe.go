package collector

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// ServiceStatus is the result of one HTTP probe.
type ServiceStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	OK        bool   `json:"ok"`
	Code      int    `json:"code"`
	LatencyMS int64  `json:"latency_ms"`
	CertDays  *int   `json:"cert_days"`
	Error     string `json:"error,omitempty"`
}

// Prober checks HTTP endpoints.
type Prober struct {
	client *http.Client
}

func NewProber(timeout time.Duration) *Prober {
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	return &Prober{client: &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives:   true,
			TLSHandshakeTimeout: timeout,
		},
	}}
}

// Probe runs a GET request and reports status, latency and certificate lifetime.
func (p *Prober) Probe(ctx context.Context, name, url string) ServiceStatus {
	res := ServiceStatus{Name: name, URL: url}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	req.Header.Set("User-Agent", "PadDeck/1.0")

	start := time.Now()
	resp, err := p.client.Do(req)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	res.Code = resp.StatusCode
	res.OK = resp.StatusCode >= 200 && resp.StatusCode < 400
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		days := int(time.Until(resp.TLS.PeerCertificates[0].NotAfter).Hours() / 24)
		res.CertDays = &days
	}
	if !res.OK {
		res.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return res
}
