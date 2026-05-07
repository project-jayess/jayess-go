package runtime

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type DNSResolverConfig struct {
	Servers  []string
	PreferGo bool
	Timeout  time.Duration
}

type DNSRecord struct {
	Address string
	Network string
}

type ParsedIP struct {
	Address  string
	Family   int
	Loopback bool
	Private  bool
}

func NewDNSResolverConfig(servers []string) DNSResolverConfig {
	return DNSResolverConfig{Servers: normalizeDNSServers(servers), PreferGo: true, Timeout: 5 * time.Second}
}

func DNSLookup(host string, config DNSResolverConfig) ([]DNSRecord, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	ctx, cancel := dnsContext(config)
	defer cancel()
	ips, err := dnsResolver(config).LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	records := make([]DNSRecord, 0, len(ips))
	for _, ip := range ips {
		records = append(records, DNSRecord{Address: ip.String(), Network: dnsIPNetwork(ip)})
	}
	return records, nil
}

func DNSReverse(address string, config DNSResolverConfig) ([]string, error) {
	if DNSIsIP(address) == 0 {
		return nil, fmt.Errorf("invalid IP address %q", address)
	}
	ctx, cancel := dnsContext(config)
	defer cancel()
	return dnsResolver(config).LookupAddr(ctx, address)
}

func DNSIsIP(address string) int {
	ip := net.ParseIP(strings.TrimSpace(address))
	if ip == nil {
		return 0
	}
	if ip.To4() != nil {
		return 4
	}
	return 6
}

func DNSParseIP(address string) (ParsedIP, bool) {
	ip := net.ParseIP(strings.TrimSpace(address))
	if ip == nil {
		return ParsedIP{}, false
	}
	return ParsedIP{
		Address:  ip.String(),
		Family:   DNSIsIP(address),
		Loopback: ip.IsLoopback(),
		Private:  ip.IsPrivate(),
	}, true
}

func dnsResolver(config DNSResolverConfig) *net.Resolver {
	if len(config.Servers) == 0 {
		return net.DefaultResolver
	}
	servers := normalizeDNSServers(config.Servers)
	return &net.Resolver{
		PreferGo: config.PreferGo || len(servers) > 0,
		Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
			var lastErr error
			dialer := net.Dialer{}
			for _, server := range servers {
				conn, err := dialer.DialContext(ctx, network, server)
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("no DNS servers configured")
			}
			return nil, lastErr
		},
	}
}

func dnsContext(config DNSResolverConfig) (context.Context, context.CancelFunc) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func normalizeDNSServers(servers []string) []string {
	normalized := make([]string, 0, len(servers))
	for _, server := range servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(server); err != nil {
			server = net.JoinHostPort(server, "53")
		}
		normalized = append(normalized, server)
	}
	return normalized
}

func dnsIPNetwork(ip net.IP) string {
	if ip.To4() != nil {
		return "ipv4"
	}
	return "ipv6"
}
