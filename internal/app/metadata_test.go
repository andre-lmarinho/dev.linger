package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBusNameMatchesMetadata keeps the compiled D-Bus name in sync with
// metadata.sh, the single source of DevLinger's identity.
func TestBusNameMatchesMetadata(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "metadata.sh"))
	if err != nil {
		t.Fatal(err)
	}
	var appid string
	for _, line := range strings.Split(string(data), "\n") {
		if value, found := strings.CutPrefix(line, "APPID="); found {
			appid = strings.TrimSpace(value)
			break
		}
	}
	if appid == "" {
		t.Fatal("metadata.sh has no APPID entry")
	}
	if appid != busName {
		t.Fatalf("busName %q does not match metadata.sh APPID %q; update both", busName, appid)
	}
}
