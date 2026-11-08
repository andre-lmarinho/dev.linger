package app

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const busName = "io.github.devlinger.DevLinger"

func Acquire() (*dbus.Conn, bool, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, false, fmt.Errorf("connect to session bus: %w", err)
	}

	reply, err := conn.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		conn.Close()
		return nil, false, fmt.Errorf("acquire application name: %w", err)
	}

	if reply != dbus.RequestNameReplyPrimaryOwner {
		conn.Close()
		return nil, false, nil
	}

	return conn, true, nil
}