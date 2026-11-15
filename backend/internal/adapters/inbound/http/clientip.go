package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ipResolver determines the client address of a request. X-Forwarded-For is
// only believed when the TCP peer is a trusted proxy.
type ipResolver struct {
	trusted []netip.Prefix
}

func (res ipResolver) isTrusted(a netip.Addr) bool {
	for _, p := range res.trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// clientIP returns the client address. If the TCP peer is a trusted proxy,
// X-Forwarded-For is walked from the right (entries appended by our own
// proxies) and the first untrusted address is the client. An unparsable
// entry stops the walk at the last trusted hop: our proxies only append
// valid addresses, so anything malformed came from beyond them.
func (res ipResolver) clientIP(r *http.Request) string {
	peer, ok := parseAddr(r.RemoteAddr)
	if !ok {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
	if !res.isTrusted(peer) {
		return peer.String()
	}
	client := peer
	values := r.Header.Values("X-Forwarded-For")
	for i := len(values) - 1; i >= 0; i-- {
		hops := strings.Split(values[i], ",")
		for j := len(hops) - 1; j >= 0; j-- {
			a, ok := parseAddr(strings.TrimSpace(hops[j]))
			if !ok {
				return client.String()
			}
			client = a
			if !res.isTrusted(a) {
				return a.String()
			}
		}
	}
	// Every hop is trusted: the leftmost one is the client.
	return client.String()
}

// parseAddr parses "ip", "ip:port" or "[ipv6]:port", dropping any IPv6 zone
// and unmapping IPv4-mapped IPv6 addresses.
func parseAddr(s string) (netip.Addr, bool) {
	a, err := netip.ParseAddr(s)
	if err != nil {
		ap, err := netip.ParseAddrPort(s)
		if err != nil {
			return netip.Addr{}, false
		}
		a = ap.Addr()
	}
	return a.WithZone("").Unmap(), true
}
