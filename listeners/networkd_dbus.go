// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 VMware, Inc.

package listeners

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/godbus/dbus/v5"

	"github.com/vmware/network-event-broker/pkg/bus"
)

const (
	dbusInterface = "org.freedesktop.network1"
	dbusPath      = "/org/freedesktop/network1"

	dbusManagerinterface = "org.freedesktop.network1.Manager"
)

type SDConnection struct {
	conn   *dbus.Conn
	object dbus.BusObject
}

func NewSDConnection() (*SDConnection, error) {
	conn, err := bus.SystemBusPrivateConn()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %v", err)
	}

	return &SDConnection{
		conn:   conn,
		object: conn.Object(dbusInterface, dbus.ObjectPath(dbusPath)),
	}, nil
}

func (c *SDConnection) Close() {
	c.conn.Close()
}

func (c *SDConnection) DBusNetworkReconfigureLink(ctx context.Context, index int) error {
	if err := c.object.CallWithContext(ctx, dbusManagerinterface+"."+"ReconfigureLink", 0, index).Err; err != nil {
		return err
	}

	return nil
}

func (c *SDConnection) DBusNetworkReload(ctx context.Context) error {
	if err := c.object.CallWithContext(ctx, dbusManagerinterface+"."+"Reload", 0).Err; err != nil {
		return err
	}

	return nil
}

func (c *SDConnection) DBusLinkDescribe(ctx context.Context) (*LinksDescribe, error) {
	var props string

	err := c.object.CallWithContext(ctx, dbusManagerinterface+"."+"Describe", 0).Store(&props)
	if err != nil {
		return nil, err
	}

	m := LinksDescribe{}
	if err := json.Unmarshal([]byte(props), &m); err != nil {
		return nil, err
	}

	return &m, nil
}