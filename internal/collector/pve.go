package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PVENode is one Proxmox node.
type PVENode struct {
	Name     string  `json:"name"`
	Online   bool    `json:"online"`
	CPU      float64 `json:"cpu"`
	MemUsed  uint64  `json:"mem_used"`
	MemTotal uint64  `json:"mem_total"`
	Uptime   uint64  `json:"uptime"`
	Status   string  `json:"status"`
}

// PVEGuest is a VM or LXC container on a node.
type PVEGuest struct {
	Node    string  `json:"node"`
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Kind    string  `json:"kind"` // qemu | lxc
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	Mem     uint64  `json:"mem"`
	MaxMem  uint64  `json:"maxmem"`
	Disk    uint64  `json:"disk"`
	MaxDisk uint64  `json:"maxdisk"`
}

type pveNode struct {
	Node   string  `json:"node"`
	Status string  `json:"status"`
	CPU    float64 `json:"cpu"`
}

type pveNodeStatus struct {
	CPU    float64 `json:"cpu"`
	Uptime uint64  `json:"uptime"`
	Memory struct {
		Used  uint64 `json:"used"`
		Total uint64 `json:"total"`
	} `json:"memory"`
}

type pveGuest struct {
	VMID    int     `json:"vmid"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	Mem     uint64  `json:"mem"`
	MaxMem  uint64  `json:"maxmem"`
	Disk    uint64  `json:"disk"`
	MaxDisk uint64  `json:"maxdisk"`
}

// PVECollector reads node and guest status through pvesh over SSH.
type PVECollector struct {
	ssh         *SSH
	displayName string
}

func NewPVECollector(ssh *SSH, displayName string) *PVECollector {
	return &PVECollector{ssh: ssh, displayName: displayName}
}

func (p *PVECollector) Collect(ctx context.Context) ([]PVENode, []PVEGuest, error) {
	out, err := p.ssh.Run(ctx, `pvesh get /nodes --output-format json 2>/dev/null`)
	if err != nil {
		return nil, nil, err
	}
	var nodes []pveNode
	if err := json.Unmarshal([]byte(out), &nodes); err != nil {
		return nil, nil, fmt.Errorf("parse pve nodes: %w", err)
	}
	// Two standalone PVE hosts both call their node "pve"; show the config name
	// instead so the panel can tell them apart.
	single := len(nodes) == 1

	var resultNodes []PVENode
	var guests []PVEGuest
	for _, n := range nodes {
		name := n.Node
		switch {
		case p.displayName == "":
		case single:
			name = p.displayName
		default:
			name = p.displayName + "/" + n.Node
		}
		status := PVENode{Name: name, Online: n.Status == "online", Status: n.Status}
		// The per-node status endpoint reports an instantaneous 0 when the
		// node is idle; the node list carries a smoothed average, so seed with
		// that and let the status call override it when non-zero.
		status.CPU = round2(n.CPU * 100)

		script := fmt.Sprintf(`echo "@@STATUS"; pvesh get /nodes/%s/status --output-format json 2>/dev/null
echo "@@QEMU"; pvesh get /nodes/%s/qemu --output-format json 2>/dev/null
echo "@@LXC"; pvesh get /nodes/%s/lxc --output-format json 2>/dev/null`, n.Node, n.Node, n.Node)
		raw, err := p.ssh.Run(ctx, script)
		if err == nil {
			secs := sections(raw)
			var st pveNodeStatus
			if json.Unmarshal([]byte(secs["STATUS"]), &st) == nil {
				if st.CPU > 0 {
					status.CPU = round2(st.CPU * 100)
				}
				status.Uptime = st.Uptime
				status.MemUsed = st.Memory.Used
				status.MemTotal = st.Memory.Total
			}
			guests = append(guests, parseGuests(secs["QEMU"], name, "qemu")...)
			guests = append(guests, parseGuests(secs["LXC"], name, "lxc")...)
		}
		resultNodes = append(resultNodes, status)
	}
	return resultNodes, guests, nil
}

func parseGuests(raw, node, kind string) []PVEGuest {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []pveGuest
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}
	out := make([]PVEGuest, 0, len(list))
	for _, g := range list {
		out = append(out, PVEGuest{
			Node:    node,
			ID:      g.VMID,
			Name:    g.Name,
			Kind:    kind,
			Status:  g.Status,
			CPU:     round2(g.CPU * 100),
			Mem:     g.Mem,
			MaxMem:  g.MaxMem,
			Disk:    g.Disk,
			MaxDisk: g.MaxDisk,
		})
	}
	return out
}
