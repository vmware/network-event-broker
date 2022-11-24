// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 VMware, Inc.

package bus

import (
	"fmt"
	"os"
	"strconv"

	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
)

const (
	DBusProperties = "org.freedesktop.DBus.Properties"

	resolveInterface      = "org.freedesktop.resolve1"
	resolveObjectPath     = "/org/freedesktop/resolve1"
	resolveSetLinkDNS     = resolveInterface + ".Manager.SetLinkDNS"
	resolveSetLinkDomains = resolveInterface + ".Manager.SetLinkDomains"
	resolveReventLink     = resolveInterface + ".Revert"

	hostnameInterface   = "org.freedesktop.hostname1"
	hostnameObjectPath  = "/org/freedesktop/hostname1"
	hostnameSetHostname = hostnameInterface + ".SetStaticHostname"
)

type DnsServer struct {
	Family  int32
	Address []byte
}

type Domain struct {
	Domain string
	Set    bool
}

func SystemBusPrivateConn() (*dbus.Conn, error) {
	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		return nil, err
	}

	methods := []dbus.Auth{dbus.AuthExternal(strconv.Itoa(os.Getuid()))}

	err = conn.Auth(methods)
	if err != nil {
		conn.Close()
		conn = nil
		return conn, err
	}

	if err = conn.Hello(); err != nil {
		conn.Close()
		conn = nil
	}

	return conn, nil
}

func SetResolveDNS(dns []DnsServer, index int) error {
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

	log.Debugf("Successfully set DNS servers ifindex='%d'", index)

	return nil
}

func SetResolveDomain(dnsDomains []Domain, index int) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %v", err)
	}
	defer conn.Close()

	log.Debugf("Setting DNS domains ifindex='%d'", index)

	obj := conn.Object(resolveInterface, resolveObjectPath)
	err = obj.Call(resolveSetLinkDomains, 0, index, dnsDomains).Store()
	if err != nil {
		return fmt.Errorf("failed to set DNS domains: %+v: %v", dnsDomains, err)
	}

	log.Debugf("Successfully set DNS domain ifindex='%d'", index)

	return nil
}

func RevertDNSLink(index int) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %v", err)
	}
	defer conn.Close()

	log.Debugf("Reverting DNS domains ifindex='%d'", index)

	obj := conn.Object(resolveInterface, resolveObjectPath)
	err = obj.Call(resolveReventLink, 0, index, 0).Store()
	if err != nil {
		return fmt.Errorf("failed to revert link='%d' DNS: %v", index, err)
	}

	log.Debugf("Successfully revert DNS ifindex='%d'", index)
	return nil
}

func SetHostname(hostname string) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system bus: %v", err)
	}
	defer conn.Close()

	log.Debugf("Setting hostname='%s'", hostname)

	obj := conn.Object(hostnameInterface, hostnameObjectPath)
	err = obj.Call(hostnameSetHostname, 0, hostname, true).Store()
	if err != nil {
		return fmt.Errorf("failed to set hostname: %w", err)
	}

	log.Debugln("Successfully set hostname")

	return nil
}
