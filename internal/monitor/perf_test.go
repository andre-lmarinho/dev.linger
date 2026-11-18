package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCatalog(b *testing.B) {
	dir := b.TempDir()
	entry := filepath.Join(dir, "node_modules", "vite", "bin", "vite.js")
	if err := os.MkdirAll(filepath.Dir(entry), 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(entry, nil, 0644); err != nil {
		b.Fatal(err)
	}
	link := filepath.Join(dir, "vite")
	if err := os.Symlink(entry, link); err != nil {
		b.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"symlink", []string{"node", link, "dev"}, "Vite"},
		{"unrecognized", []string{"node", entry, "build"}, ""},
	} {
		b.Run(tc.name, func(b *testing.B) {
			if got := catalog(tc.args, dir); got != tc.want {
				b.Fatalf("got %q, want %q", got, tc.want)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				catalog(tc.args, dir)
			}
		})
	}
}
