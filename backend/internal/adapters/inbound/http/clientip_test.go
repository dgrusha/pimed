package httpapi

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIP(t *testing.T) {
	res := ipResolver{trusted: []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("172.18.0.0/16"),
	}}
	tests := []struct {
		name   string
		remote string
		xff    []string
		want   string
	}{
		{"no proxy", "192.0.2.1:1234", nil, "192.0.2.1"},
		{"untrusted peer spoofing XFF", "192.0.2.1:1234", []string{"10.0.0.1"}, "192.0.2.1"},
		{"trusted peer without XFF", "127.0.0.1:1234", nil, "127.0.0.1"},
		{"trusted peer with XFF", "127.0.0.1:1234", []string{"198.51.100.4"}, "198.51.100.4"},
		{"client-supplied entries ignored", "127.0.0.1:1234", []string{"1.1.1.1, 2.2.2.2, 198.51.100.4"}, "198.51.100.4"},
		{"chained proxies", "127.0.0.1:1234", []string{"6.6.6.6, 198.51.100.4, 172.18.0.5"}, "198.51.100.4"},
		{"multiple XFF headers", "127.0.0.1:1234", []string{"6.6.6.6", "198.51.100.4, 172.18.0.5"}, "198.51.100.4"},
		{"all hops trusted", "127.0.0.1:1234", []string{"172.18.0.9, 172.18.0.5"}, "172.18.0.9"},
		{"garbage stops at last trusted hop", "127.0.0.1:1234", []string{"not-an-ip, 172.18.0.5"}, "172.18.0.5"},
		{"garbage only", "127.0.0.1:1234", []string{"<script>"}, "127.0.0.1"},
		{"empty entry", "127.0.0.1:1234", []string{"198.51.100.4, "}, "127.0.0.1"},
		{"entry with port", "127.0.0.1:1234", []string{"198.51.100.4:5555"}, "198.51.100.4"},
		{"IPv6 peer trusted", "[::1]:1234", []string{"2001:db8::7"}, "2001:db8::7"},
		{"IPv6 bracketed entry", "[::1]:1234", []string{"[2001:db8::7]:443"}, "2001:db8::7"},
		{"IPv6 untrusted peer", "[2001:db8::1]:1234", []string{"198.51.100.4"}, "2001:db8::1"},
		{"IPv4-mapped peer", "[::ffff:127.0.0.1]:1234", []string{"198.51.100.4"}, "198.51.100.4"},
		{"IPv6 zone dropped", "[fe80::1%eth0]:1234", nil, "fe80::1"},
		{"unparsable peer", "garbage", []string{"198.51.100.4"}, "garbage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remote
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := res.clientIP(r); got != tt.want {
				t.Fatalf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}

	// Trusting nobody always yields the peer.
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "127.0.0.1:1"
	r.Header.Set("X-Forwarded-For", "198.51.100.4")
	if got := (ipResolver{}).clientIP(r); got != "127.0.0.1" {
		t.Fatalf("no trusted proxies: %q", got)
	}
}
