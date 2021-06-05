//SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 VMware, Inc.
//

package bus

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/network-event-broker/pkg/log"
)

const (
	resolveInterface  = "org.freedesktop.resolve1"
	resolveObjectPath = "/org/freedesktop/resolve1"
	resolveSetLinkDNS = resolveInterface + ".Manager.SetLinkDNS"
)

type DnsServer struct {
	Family  int32
	Address []byte
}

func SetResolve(dns []DnsServer, index int) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %v", err)
	}
	defer conn.Close()

	log.Debugf("Setting DNS servers ifindex='%d'", index)

	obj := conn.Object(resolveInterface, resolveObjectPath)
	err = obj.Call(resolveSetLinkDNS, 0, index, dns).Store()
	if err != nil {
		return fmt.Errorf("failed to set DNS servers: %w", err)
	}

	log.Debugln("Successfully set DNS servers")

	return nil
}
