// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 VMware, Inc.

package listeners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jaypipes/ghw"

	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vmware/network-event-broker/pkg/bus"
	"github.com/vmware/network-event-broker/pkg/conf"
	"github.com/vmware/network-event-broker/pkg/configfile"
	"github.com/vmware/network-event-broker/pkg/network"
	"github.com/vmware/network-event-broker/pkg/system"
)

const (
	networkInterface  = "org.freedesktop.network1"
	networkObjectPath = "/org/freedesktop/network1"

	networkInterfaceLink       = "org.freedesktop.network1.Link"
	networkInterfaceLinkEscape = networkObjectPath + "/link/_3"

	defaultRequestTimeout = 5 * time.Second
)

type LinkDescribe struct {
	AddressState     string   `json:"AddressState"`
	AlternativeNames []string `json:"AlternativeNames"`
	CarrierState     string   `json:"CarrierState"`
	Driver           string   `json:"Driver"`
	IPv4AddressState string   `json:"IPv4AddressState"`
	IPv6AddressState string   `json:"IPv6AddressState"`
	Index            int      `json:"Index"`
	LinkFile         string   `json:"LinkFile"`
	Model            string   `json:"Model"`
	Name             string   `json:"Name"`
	OnlineState      string   `json:"OnlineState"`
	OperationalState string   `json:"OperationalState"`
	Path             string   `json:"Path"`
	SetupState       string   `json:"SetupState"`
	Type             string   `json:"Type"`
	Vendor           string   `json:"Vendor"`
	Manufacturer     string   `json:"Manufacturer"`
	NetworkFile      string   `json:"NetworkFile,omitempty"`
	DNS              []string `json:"DNS"`
	Domains          []string `json:"Domains"`
	NTP              []string `json:"NTP"`
}

type LinksDescribe struct {
	Interfaces []LinkDescribe
}

func fillOneLink(link netlink.Link) *LinkDescribe {
	l := LinkDescribe{
		Index: link.Attrs().Index,
		Name:  link.Attrs().Name,
		Type:  link.Attrs().EncapType,
	}

	l.AddressState, _ = ParseLinkAddressState(link.Attrs().Index)
	l.IPv4AddressState, _ = ParseLinkIPv4AddressState(link.Attrs().Index)
	l.IPv6AddressState, _ = ParseLinkIPv6AddressState(link.Attrs().Index)
	l.CarrierState, _ = ParseLinkCarrierState(link.Attrs().Index)
	l.OnlineState, _ = ParseLinkOnlineState(link.Attrs().Index)
	l.OperationalState, _ = ParseLinkOperationalState(link.Attrs().Index)
	l.SetupState, _ = ParseLinkSetupState(link.Attrs().Index)
	l.NetworkFile, _ = ParseLinkNetworkFile(link.Attrs().Index)
	l.DNS, _ = ParseLinkDNS(link.Attrs().Index)
	l.Domains, _ = ParseLinkDomains(link.Attrs().Index)
	l.NTP, _ = ParseLinkNTP(link.Attrs().Index)

	c, err := configfile.ParseKeyFromSectionString(path.Join("/sys/class/net", link.Attrs().Name, "device/uevent"), "", "PCI_SLOT_NAME")
	if err == nil {
		pci, err := ghw.PCI()
		if err == nil {
			dev := pci.GetDevice(c)

			l.Model = dev.Product.Name
			l.Vendor = dev.Vendor.Name
			l.Path = "pci-" + dev.Address
		}
	}

	driver, err := configfile.ParseKeyFromSectionString(path.Join("/sys/class/net", link.Attrs().Name, "device/uevent"), "", "DRIVER")
	if err == nil {
		l.Driver = driver
	}

	return &l
}

func buildLinkMessageFallback(link string) (*LinkDescribe, error) {
	l, err := netlink.LinkByName(link)
	if err != nil {
		return nil, err
	}

	return fillOneLink(l), nil
}

func acquireLink(link string) (*LinkDescribe, error) {
	c, err := NewSDConnection()
	if err != nil {
		log.Errorf("Failed to establish connection to the system bus: %s", err)
		return nil, err
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	links, err := c.DBusLinkDescribe(ctx)
	if err != nil {
		return buildLinkMessageFallback(link)
	}

	for _, l := range links.Interfaces {
		if l.Name == link {
			return &l, nil
		}
	}

	return nil, errors.New("not found")
}

func executeNetworkdLinkStateScripts(link string, index int, k string, v string, c *conf.Config) error {
	scriptDirs, err := system.ReadAllScriptDirs(conf.ConfPath)
	if err != nil {
		log.Errorf("Failed to find any scripts in conf dir: %+v", err)
		return err
	}

	for _, d := range scriptDirs {
		stateDir := v + ".d"

		if stateDir == d {
			scripts, err := system.ReadAllScriptInConfDir(path.Join(conf.ConfPath, d))
			if err != nil {
				log.Errorf("Failed to read script dir '%s'", path.Join(conf.ConfPath, d))
				continue
			}

			path.Join(conf.ConfPath, d)
			linkNameEnvArg := "LINK=" + link
			linkIndexEnvArg := "LINKINDEX=" + strconv.Itoa(index)
			linkStateEnvArg := k + "=" + v

			if len(scripts) <= 0 {
				continue
			}

			leaseFile := path.Join(conf.NetworkdLeasePath, strconv.Itoa(index))
			leaseLines, err := system.ReadLines(leaseFile)
			if err != nil {
				log.Debugf("Failed to read lease file of link='%+v'", link, err)
			}

			var leaseArg string
			if len(leaseLines) > 0 {
				leaseArg = "DHCP_LEASE="
				leaseArg += strings.Join(leaseLines, " ")
			}

			var jsonData string
			if c.Network.EmitJSON {
				m, err := acquireLink(link)
				if err == nil {
					j, _ := json.Marshal(m)
					jsonData = "JSON=" + string(j)

					log.Debugf("JSON : %v\n", jsonData)
				}
			}

			for _, s := range scripts {
				script := path.Join(conf.ConfPath, d, s)

				log.Debugf("Executing script '%s' in dir='%v' for link='%s'", script, d, link)

				cmd := exec.Command(script)
				cmd.Env = append(os.Environ(),
					linkNameEnvArg,
					linkNameEnvArg,
					linkIndexEnvArg,
					linkStateEnvArg,
					leaseArg,
				)

				if c.Network.EmitJSON {
					cmd.Env = append(cmd.Env, jsonData)
				}

				if err := cmd.Run(); err != nil {
					log.Errorf("Failed to execute script='%s': %v", script, err)
					continue
				}

				log.Debugf("Successfully executed script '%s' in dir='%v' for link='%s'", script, d, link)
			}
		}
	}

	return nil
}

func executeNetworkdManagerScripts(k string, v string) error {
	managerStatePath := path.Join(conf.ConfPath, conf.ManagerStateDir)

	scripts, err := system.ReadAllScriptInConfDir(managerStatePath)
	if err != nil {
		log.Errorf("Failed to read script dir '%s'", managerStatePath)
		return nil
	}

	for _, s := range scripts {
		script := path.Join(managerStatePath, s)

		log.Debugf("Executing script '%s' in dir='%s'", script, managerStatePath)

		managerStateEnvArg := k + "=" + v
		cmd := exec.Command(script)
		cmd.Env = append(os.Environ(),
			managerStateEnvArg,
			managerStateEnvArg,
		)

		if err := cmd.Run(); err != nil {
			log.Errorf("Failed to execute script='%s': %+v", script, err)
			continue
		}

		log.Debugf("Successfully executed script '%s' in dir='%v' for manager state", script, managerStatePath)
	}

	return nil
}

func processDBusLinkMessage(n *network.Network, v *dbus.Signal, c *conf.Config) error {
	if !strings.HasPrefix(string(v.Path), networkInterfaceLinkEscape) {
		return nil
	}

	strIndex := strings.TrimPrefix(string(v.Path), networkInterfaceLinkEscape)
	index, err := strconv.Atoi(strIndex)
	if err != nil {
		log.Errorf("Failed to convert ifindex=\"%s\" to integer: %+v", strIndex, err)
		return nil
	}

	log.Debugf("Received DBus signal from systemd-networkd for ifindex='%d'", index)

	linkState := v.Body[1].(map[string]dbus.Variant)
	for k, v := range linkState {
		switch k {
		case "OperationalState":
			{
				s := strings.Trim(v.String(), "\"")

				log.Debugf("Link='%s' ifindex='%d' changed state '%s'='%s'", n.LinksByIndex[index], index, k, s)

				if c.Network.Links != "" {
					if strings.Contains(c.Network.Links, n.LinksByIndex[index]) {
						executeNetworkdLinkStateScripts(n.LinksByIndex[index], index, k, s, c)
					}
				} else {
					executeNetworkdLinkStateScripts(n.LinksByIndex[index], index, k, s, c)
				}

				if s == "routable" && strings.Contains(c.Network.RoutingPolicyRules, n.LinksByIndex[index]) {
					network.ConfigureNetwork(n.LinksByIndex[index], n)
				}
			}
		}
	}

	return nil
}

func processDBusManagerMessage(n *network.Network, v *dbus.Signal) error {
	state := v.Body[1].(map[string]dbus.Variant)

	for k, v := range state {
		s := strings.Trim(v.String(), "\"")

		log.Debugf("Manager chaged state '%v='%v'", k, s)

		executeNetworkdManagerScripts(k, s)
	}

	return nil
}

func WatchNetworkd(n *network.Network, c *conf.Config, finished chan bool) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	opts := []dbus.MatchOption{
		dbus.WithMatchSender(networkInterface),
		dbus.WithMatchInterface(bus.DBusProperties),
		dbus.WithMatchMember("PropertiesChanged"),
	}

	if err := conn.AddMatchSignal(opts...); err != nil {
		log.Errorf("Failed to add match signal for '%s`: %+v", networkInterface, err)
		return err
	}

	log.Infoln("Listening to systemd-networkd DBus events")

	sigChannel := make(chan *dbus.Signal, 512)
	conn.Signal(sigChannel)

	for v := range sigChannel {
		w := fmt.Sprintf("%v", v.Body[0])

		if strings.HasPrefix(w, networkInterfaceLink) {
			log.Debugf("Received Link DBus signal from systemd-networkd'")

			go processDBusLinkMessage(n, v, c)

		} else if strings.HasPrefix(w, "org.freedesktop.network1.Manager") {
			log.Debugf("Received Manager DBus signal from systemd-networkd'")

			go processDBusManagerMessage(n, v)
		}
	}

	finished <- true

	return nil
}
