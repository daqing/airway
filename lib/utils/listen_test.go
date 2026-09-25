package utils

import "testing"

func TestListenHostPort(t *testing.T) {
	for _, tc := range []struct {
		listen string
		host   string
		port   string
		ok     bool
	}{
		{":1900", "", "1900", true},
		{"0.0.0.0:1905", "", "1905", true},
		{"[::]:1905", "", "1905", true},
		{"127.0.0.1:1905", "127.0.0.1", "1905", true},
		{"192.168.1.5:1905", "192.168.1.5", "1905", true},
		{"[::1]:1905", "::1", "1905", true},
		{"1905", "", "", false},
	} {
		host, port, ok := ListenHostPort(tc.listen)
		if host != tc.host || port != tc.port || ok != tc.ok {
			t.Fatalf("ListenHostPort(%q) = %q, %q, %v; want %q, %q, %v", tc.listen, host, port, ok, tc.host, tc.port, tc.ok)
		}
	}
}

func TestListenAddress(t *testing.T) {
	for _, tc := range []struct {
		name   string
		shell  map[string]string
		loaded map[string]string
		want   string
	}{
		{
			name:   "listen from the loaded environment",
			loaded: map[string]string{"LISTEN": "0.0.0.0:1905"},
			want:   "0.0.0.0:1905",
		},
		{
			name:   "airway prefix preferred",
			loaded: map[string]string{"AIRWAY_LISTEN": ":1906", "LISTEN": "0.0.0.0:1905"},
			want:   ":1906",
		},
		{
			name:   "shell listen wins over loaded",
			shell:  map[string]string{"LISTEN": "[::1]:1907"},
			loaded: map[string]string{"LISTEN": "0.0.0.0:1905"},
			want:   "[::1]:1907",
		},
		{
			name:   "blank value falls back to the default",
			shell:  map[string]string{"LISTEN": "   "},
			loaded: map[string]string{"LISTEN": ""},
			want:   DefaultListen,
		},
		{
			name: "nothing configured",
			want: DefaultListen,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withShellEnv(t, tc.shell)

			for _, key := range []string{"AIRWAY_LISTEN", "LISTEN"} {
				clearEnv(key, t)
			}
			for key, value := range tc.loaded {
				setEnv(key, value, t)
			}

			if got := ListenAddress(); got != tc.want {
				t.Fatalf("ListenAddress() = %q, want %q", got, tc.want)
			}
		})
	}
}
