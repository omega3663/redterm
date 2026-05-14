package context

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	nmapHostRE    = regexp.MustCompile(`Nmap scan report for (.+)`)
	nmapPortRE    = regexp.MustCompile(`^(\d+)/(tcp|udp)\s+open\s+(\S+)(?:\s+(.+))?`)
	netexecCredRE = regexp.MustCompile(`\[\+\].*\\([^\\:]+):(\S+)`)
	bhdDomainRE   = regexp.MustCompile(`\[\*\]\s+Domain is\s+(\S+)`)
	bhdFoundRE    = regexp.MustCompile(`\[\*\]\s+Found (\d+) (computers|users)`)
)

type HostEntry struct {
	IP       string
	Hostname string
	Ports    []string
}

type CredEntry struct {
	Type   string
	Value  string
	Source string
}

type DomainInfo struct {
	Name      string
	UserCount string
	CompCount string
}

type findings struct {
	mu          sync.Mutex
	hosts       map[string]*HostEntry
	hostOrder   []string // insertion order for stable output
	creds       []CredEntry
	domain      DomainInfo
	currentHost string
}

func newFindings() *findings {
	return &findings{hosts: make(map[string]*HostEntry)}
}

func (f *findings) parseLine(line string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// nmap: host report header
	if m := nmapHostRE.FindStringSubmatch(line); m != nil {
		target := strings.TrimSpace(m[1])
		ip, hostname := target, ""
		if i := strings.Index(target, " ("); i != -1 {
			ip = target[:i]
			hostname = strings.Trim(target[i+2:], "()")
		}
		if _, ok := f.hosts[ip]; !ok {
			f.hosts[ip] = &HostEntry{IP: ip, Hostname: hostname}
			f.hostOrder = append(f.hostOrder, ip)
		}
		f.currentHost = ip
		return
	}

	// nmap: open port line (requires a preceding host header)
	if f.currentHost != "" {
		if m := nmapPortRE.FindStringSubmatch(line); m != nil {
			detail := strings.TrimSpace(m[4])
			portStr := m[1] + "/" + m[2] + " " + m[3]
			if detail != "" {
				portStr += " — " + detail
			}
			h := f.hosts[f.currentHost]
			h.Ports = append(h.Ports, portStr)
			return
		}
	}

	// netexec: credential success (DOMAIN\user:pass)
	if m := netexecCredRE.FindStringSubmatch(line); m != nil {
		f.creds = append(f.creds, CredEntry{
			Type:   "credential",
			Value:  m[1] + ":" + m[2],
			Source: "netexec",
		})
		return
	}

	// bloodhound-python
	if m := bhdDomainRE.FindStringSubmatch(line); m != nil {
		f.domain.Name = m[1]
	}
	if m := bhdFoundRE.FindStringSubmatch(line); m != nil {
		switch m[2] {
		case "users":
			f.domain.UserCount = m[1]
		case "computers":
			f.domain.CompCount = m[1]
		}
	}
}

func (f *findings) empty() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.hosts) == 0 && len(f.creds) == 0 && f.domain.Name == ""
}

func (f *findings) format() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.hosts) == 0 && len(f.creds) == 0 && f.domain.Name == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("[STRUCTURED FINDINGS]\n")

	if len(f.hosts) > 0 {
		sb.WriteString("HOSTS:\n")
		for _, ip := range f.hostOrder {
			h := f.hosts[ip]
			entry := "  " + h.IP
			if h.Hostname != "" {
				entry += " (" + h.Hostname + ")"
			}
			sb.WriteString(entry + "\n")
			for _, p := range h.Ports {
				sb.WriteString("    " + p + "\n")
			}
		}
	}

	if len(f.creds) > 0 {
		sb.WriteString("CREDENTIALS:\n")
		for _, c := range f.creds {
			sb.WriteString(fmt.Sprintf("  [%s] %s via %s\n", c.Type, c.Value, c.Source))
		}
	}

	if f.domain.Name != "" {
		sb.WriteString("DOMAIN:\n")
		sb.WriteString("  Name: " + f.domain.Name + "\n")
		if f.domain.UserCount != "" {
			sb.WriteString("  Users: " + f.domain.UserCount + "\n")
		}
		if f.domain.CompCount != "" {
			sb.WriteString("  Computers: " + f.domain.CompCount + "\n")
		}
	}

	sb.WriteString("[END STRUCTURED FINDINGS]\n")
	return sb.String()
}
