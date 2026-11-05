package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadListeners(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "net"), 0755); err != nil {
		t.Fatal(err)
	}
	data := "  sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n  0: 0100007F:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 12345\n  1: 0100007F:0BB9 00000000:0000 01 00000000:00000000 00:00000000 00000000 1000 0 12346\n"
	if err := os.WriteFile(filepath.Join(dir, "net", "tcp"), []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readListeners(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].address != "127.0.0.1" || got[0].port != 3000 || got[0].inode != "12345" {
		t.Fatalf("unexpected listeners: %#v", got)
	}
}

func TestDecodeIPv6(t *testing.T) {
	if got := decodeAddress("00000000000000000000000001000000"); got != "::1" {
		t.Fatal(got)
	}
}

func TestPublishedMatchesAddressAndPort(t *testing.T) {
	c := container{}
	c.Ports = append(c.Ports, struct {
		IP         string `json:"IP"`
		PublicPort uint16 `json:"PublicPort"`
		Type       string `json:"Type"`
	}{"127.0.0.1", 3000, "tcp"})
	if !published([]container{c}, listener{address: "127.0.0.1", port: 3000}) || published([]container{c}, listener{address: "0.0.0.0", port: 3000}) {
		t.Fatal("incorrect published port match")
	}
}
