package middleware

import "testing"

func TestIsHostAllowed(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		allowedHosts []string
		want         bool
	}{
		// Empty allowedHosts (backwards compatible)
		{
			name:         "empty allowed hosts returns true",
			host:         "example.com",
			allowedHosts: []string{},
			want:         true,
		},

		// IPv4 tests
		{
			name:         "IPv4 exact match",
			host:         "example.com:8080",
			allowedHosts: []string{"example.com:8080"},
			want:         true,
		},
		{
			name:         "IPv4 host without port matches allowed with port",
			host:         "example.com",
			allowedHosts: []string{"example.com:8080"},
			want:         true,
		},
		{
			name:         "IPv4 host with port matches allowed without port",
			host:         "example.com:8080",
			allowedHosts: []string{"example.com"},
			want:         true,
		},
		{
			name:         "IPv4 localhost with port",
			host:         "localhost:3000",
			allowedHosts: []string{"localhost"},
			want:         true,
		},

		// IPv6 tests
		{
			name:         "IPv6 loopback with port",
			host:         "[::1]:8080",
			allowedHosts: []string{"[::1]:8080"},
			want:         true,
		},
		{
			name:         "IPv6 without port matches allowed with port",
			host:         "::1",
			allowedHosts: []string{"[::1]:8080"},
			want:         true,
		},
		{
			name:         "IPv6 with port matches allowed without port",
			host:         "[::1]:8080",
			allowedHosts: []string{"::1"},
			want:         true,
		},
		{
			name:         "IPv6 full address with port",
			host:         "[2001:0db8:85a3::8a2e:0370:7334]:443",
			allowedHosts: []string{"2001:0db8:85a3::8a2e:0370:7334"},
			want:         true,
		},
		{
			name:         "IPv6 link-local with zone",
			host:         "[fe80::1%lo0]:8080",
			allowedHosts: []string{"fe80::1%lo0"},
			want:         true,
		},

		// Case insensitivity
		{
			name:         "case insensitive match",
			host:         "Example.COM:8080",
			allowedHosts: []string{"example.com"},
			want:         true,
		},

		// Whitespace handling
		{
			name:         "host with whitespace",
			host:         "  example.com:8080  ",
			allowedHosts: []string{"example.com"},
			want:         true,
		},
		{
			name:         "allowed host with whitespace",
			host:         "example.com:8080",
			allowedHosts: []string{"  example.com  "},
			want:         true,
		},

		// Multiple allowed hosts
		{
			name:         "match second in list",
			host:         "app.example.com",
			allowedHosts: []string{"example.com", "app.example.com", "api.example.com"},
			want:         true,
		},

		// Rejection cases
		{
			name:         "no match returns false",
			host:         "evil.com",
			allowedHosts: []string{"example.com", "app.example.com"},
			want:         false,
		},
		{
			name:         "subdomain mismatch",
			host:         "sub.example.com",
			allowedHosts: []string{"example.com"},
			want:         false,
		},
		{
			name:         "IPv6 different address",
			host:         "[::2]:8080",
			allowedHosts: []string{"[::1]:8080"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHostAllowed(tt.host, tt.allowedHosts)
			if got != tt.want {
				t.Errorf("IsHostAllowed(%q, %v) = %v, want %v",
					tt.host, tt.allowedHosts, got, tt.want)
			}
		})
	}
}
