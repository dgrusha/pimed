package config

import (
	"log/slog"
	"net/netip"
	"slices"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	defaultProxies := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32"), netip.MustParsePrefix("::1/128")}
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(Config) bool
	}{
		{"defaults", nil, false, func(c Config) bool {
			return c.Addr == "127.0.0.1:8080" && c.DataDir == "/data" && c.DBPath == "/var/lib/pimed/pimed.db" &&
				c.CookieSecure && c.SessionTTL == 720*time.Hour && c.LogLevel == slog.LevelInfo &&
				slices.Equal(c.TrustedProxies, defaultProxies)
		}},
		{"overrides", map[string]string{
			"PIMED_ADDR": ":9000", "PIMED_DATA_DIR": "./tmp/data", "PIMED_DB_PATH": "./tmp/p.db",
			"PIMED_COOKIE_SECURE": "false", "PIMED_SESSION_TTL": "2h", "PIMED_LOG_LEVEL": "DEBUG",
			"PIMED_TRUSTED_PROXIES": " 172.18.0.0/16 , 10.0.0.1,fd00::1/64 ",
		}, false, func(c Config) bool {
			return c.Addr == ":9000" && c.DataDir == "./tmp/data" && c.DBPath == "./tmp/p.db" &&
				!c.CookieSecure && c.SessionTTL == 2*time.Hour && c.LogLevel == slog.LevelDebug &&
				slices.Equal(c.TrustedProxies, []netip.Prefix{
					netip.MustParsePrefix("172.18.0.0/16"),
					netip.MustParsePrefix("10.0.0.1/32"),
					netip.MustParsePrefix("fd00::/64"),
				})
		}},
		{"trust no proxy", map[string]string{"PIMED_TRUSTED_PROXIES": ""}, false, func(c Config) bool {
			return len(c.TrustedProxies) == 0
		}},
		{"bad proxy", map[string]string{"PIMED_TRUSTED_PROXIES": "127.0.0.1,localhost"}, true, nil},
		{"bad cidr", map[string]string{"PIMED_TRUSTED_PROXIES": "10.0.0.0/33"}, true, nil},
		{"bad bool", map[string]string{"PIMED_COOKIE_SECURE": "nah"}, true, nil},
		{"bad ttl", map[string]string{"PIMED_SESSION_TTL": "forever"}, true, nil},
		{"ttl too small", map[string]string{"PIMED_SESSION_TTL": "1s"}, true, nil},
		{"bad level", map[string]string{"PIMED_LOG_LEVEL": "loud"}, true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(func(k string) (string, bool) { v, ok := tt.env[k]; return v, ok })
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v", err)
			}
			if tt.check != nil && !tt.check(cfg) {
				t.Fatalf("unexpected config %+v", cfg)
			}
		})
	}
}
