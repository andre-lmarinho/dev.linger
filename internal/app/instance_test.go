package app

import (
	"os"
	"testing"
)

func TestSingleInstance(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}

	first, acquired, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("first instance should acquire the name")
	}
	defer first.Close()

	second, acquired, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if second != nil {
		defer second.Close()
	}
	if acquired {
		t.Fatal("second instance must not acquire the name")
	}

	first.Close()

	third, acquired, err := Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("name should be released after first instance closes")
	}
	defer third.Close()
}