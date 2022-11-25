// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 VMware, Inc.

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

	"github.com/godbus/dbus/v5"
	"github.com/jaypipes/ghw"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/vmware/network-event-broker/pkg/bus"
	"github.com/vmware/network-event-broker/pkg/conf"
	"github.com/vmware/network-event-broker/pkg/configfile"
	"github.com/vmware/network-event-broker/pkg/network"
	"github.com/vmware/network-event-broker/pkg/parser"
	"github.com/vmware/network-event-broker/pkg/system"
)

const (
	networkInterface  = "org.freedesktop.network1"
	networkObjectPath = "/org/freedesktop/network1"

	networkInterfaceLink       = "org.freedesktop.network1.Link"
	networkInterfaceLinkEscape = networkObjectPath + "/link/_3"

	defaultRequestTimeout = 5 * time.Second
)

type Route struct {
	Scope int `json:"Scope"`
	Dst   struct {
		IP   string `json:"IP"`
		Mask int    `json:"Mask"`
	} `json:"Dst"`
	Src       string   `json:"Src"`
	Gw        string   `json:"Gw"`
	MultiPath string   `json:"MultiPath"`
	Protocol  int      `json:"Protocol"`
	Priority  int      `json:"Priority"`
	Table     int      `json:"Table"`
	Type      int      `json:"Type"`
	Tos       int      `json:"Tos"`
	Flags     []string `json:"Flags"`
	MPLSDst   string   `json:"MPLSDst"`
	NewDst    string   `json:"NewDst"`
	Encap     string   `json:"Encap"`
	Mtu       int      `json:"MTU"`
	AdvMSS    int      `json:"AdvMSS"`
	Hoplimit  int      `json:"Hoplimit"`
}

type Address struct {
	IP          string `json:"IP"`
	Family      string   `json:"Family"`
	Mask        int    `json:"Mask"`
	Label       string `json:"Label"`
	Flags       int    `json:"Flags"`
	Scope       int    `json:"Scope"`
	Peer        string `json:"Peer"`
	Broadcast   string `json:"Broadcast"`
	PreferedLft int    `json:"PreferedLft"`
	ValidLft    int    `json:"ValidLft"`
}

type LinkDescribe struct {
	Index            int                     `json:"Index"`
	Mtu              int                     `json:"MTU"`
	TxQLen           int                     `json:"TxQLen"`
	Name             string                  `json:"Name"`
	AlternativeNames string                  `json:"AlternativeNames"`
	HardwareAddr     string                  `json:"HardwareAddr"`
	Flags            string                  `json:"Flags"`
	RawFlags         uint32                  `json:"RawFlags"`
	ParentIndex      int                     `json:"ParentIndex"`
	MasterIndex      int                     `json:"MasterIndex"`
	Namespace        string                  `json:"Namespace"`
	Alias            string                  `json:"Alias"`
	Statistics       *netlink.LinkStatistics `json:"Statistics"`

	Promisc int `json:"Promisc"`
	Xdp     struct {
		Fd       int  `json:"Fd"`
		Attached bool `json:"Attached"`
		Flags    int  `json:"Flags"`
		ProgID   int  `json:"ProgId"`
	} `json:"Xdp"`
	EncapType       string `json:"EncapType"`
	Protinfo        string `json:"Protinfo"`
	OperState       string `json:"OperState"`
	NetNsID         int    `json:"NetNsID"`
	NumTxQueues     int    `json:"NumTxQueues"`
	NumRxQueues     int    `json:"NumRxQueues"`
	GSOMaxSize      uint32 `json:"GSOMaxSize"`
	GSOMaxSegs      uint32 `json:"GSOMaxSegs"`
	Group           uint32 `json:"Group"`
	Slave           string `json:"Slave"`
	KernelOperState string `json: "KernelOperState"`

	AddressState     string `json:"AddressState"`
	CarrierState     string `json:"CarrierState"`
	Driver           string `json:"Driver"`
	IPv4AddressState string `json:"IPv4AddressState"`
	IPv6AddressState string `json:"IPv6AddressState"`

	LinkFile         string   `json:"LinkFile"`
	Model            string   `json:"Model"`
	OnlineState      string   `json:"OnlineState"`
	OperationalState string   `json:"OperationalState"`
	Path             string   `json:"Path"`
	SetupState       string   `json:"SetupState"`
	Type             string   `json:"Type"`
	Vendor           string   `json:"Vendor"`
	ProductID        string   `json:"ProductID"`
	Manufacturer     string   `json:"Manufacturer"`
	NetworkFile      string   `json:"NetworkFile,omitempty"`
	DNS              []string `json:"DNS"`
	Domains          []string `json:"Domains"`
	NTP              []string `json:"NTP"`

	Addresses []Address `json:"Address"`
	Routes    []Route   `json:"Routes"`
}

type LinksDescribe struct {
	Interfaces []LinkDescribe
}

func fillOneRoute(rt *netlink.Route) Route {
	route := Route{
		Scope:    int(rt.Scope),
		Protocol: rt.Protocol,
		Priority: rt.Priority,
		Table:    rt.Table,
		Type:     rt.Type,
		Tos:      rt.Tos,
		Mtu:      rt.MTU,
		AdvMSS:   rt.AdvMSS,
		Hoplimit: rt.Hoplimit,
	}

	if rt.Gw != nil {
		route.Gw = rt.Gw.String()
	}

	if rt.Src != nil {
		route.Src = rt.Src.String()
	}

	if rt.Dst != nil {
		route.Dst.IP = rt.Dst.IP.String()
		route.Dst.Mask, _ = rt.Dst.Mask.Size()
	}

	if rt.Flags != 0 {
		route.Flags = rt.ListFlags()
	}

	return route
}

func fillOneAddress(a *netlink.Addr) Address {
	addr := Address{
		IP:          a.IP.String(),
		Label:       a.Label,
		Scope:       a.Scope,
		Flags:       a.Flags,
		PreferedLft: a.PreferedLft,
		ValidLft:    a.ValidLft,
	}

	addr.Family = parser.IP4or6(a.IP.String())

	addr.Mask, _ = a.Mask.Size()
	if a.Peer != nil {
		addr.Peer = a.Peer.String()
	}

	if a.Broadcast != nil {
		addr.Broadcast = a.Broadcast.String()
	}

	return addr
}

func fillOneLink(link netlink.Link) *LinkDescribe {
	l := LinkDescribe{
		Type:            link.Attrs().EncapType,
		KernelOperState: link.Attrs().OperState.String(),
		Index:           link.Attrs().Index,
		Mtu:             link.Attrs().MTU,
		TxQLen:          link.Attrs().TxQLen,
		Name:            link.Attrs().Name,
		HardwareAddr:    link.Attrs().HardwareAddr.String(),
		RawFlags:        link.Attrs().RawFlags,
		ParentIndex:     link.Attrs().ParentIndex,
		MasterIndex:     link.Attrs().MasterIndex,
		Alias:           link.Attrs().Alias,
		EncapType:       link.Attrs().EncapType,
		OperState:       link.Attrs().OperState.String(),
		NetNsID:         link.Attrs().NetNsID,
		NumTxQueues:     link.Attrs().NumTxQueues,
		NumRxQueues:     link.Attrs().NumRxQueues,
		GSOMaxSize:      link.Attrs().GSOMaxSize,
		GSOMaxSegs:      link.Attrs().GSOMaxSegs,
		Group:           link.Attrs().Group,
		Statistics:      link.Attrs().Statistics,
		Promisc:         link.Attrs().Promisc,
		Flags:           link.Attrs().Flags.String(),
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

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return &l
	}

	for _, a := range addrs {
		l.Addresses = append(l.Addresses, fillOneAddress(&a))
	}

	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil
	}

	for _, rt := range routes {
		if rt.LinkIndex != link.Attrs().Index {
			continue
		}

		l.Routes = append(l.Routes, fillOneRoute(&rt))
	}

	c, err := configfile.ParseKeyFromSectionString(path.Join("/sys/class/net", link.Attrs().Name, "device/uevent"), "", "PCI_SLOT_NAME")
	if err == nil {
		pci, err := ghw.PCI()
		if err == nil {
			dev := pci.GetDevice(c)

			l.Model = dev.Product.Name
			l.Vendor = dev.Vendor.Name
			l.Path = "pci-" + dev.Address
			l.Driver = dev.Driver
			l.ProductID = dev.Product.ID
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

			if len(scripts) <= 0 {
				log.Debugf("No script in '%+v'", d)
				continue
			}

			path.Join(conf.ConfPath, d)
			linkNameEnvArg := "LINK=" + link
			linkIndexEnvArg := "LINKINDEX=" + strconv.Itoa(index)
			linkStateEnvArg := k + "=" + v

			leaseFile := path.Join(conf.NetworkdLeasePath, strconv.Itoa(index))
			leaseLines, err := system.ReadLines(leaseFile)
			if err != nil {
				log.Debugf("Failed to read lease file of link='%+v'", link, err)
				continue
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
		log.Errorf("Failed to read script dir '%s': %+v", managerStatePath, err)
		return err
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

		log.Debugf("Manager changed state '%v='%v'", k, s)

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

	log.Infoln("Listening to 'systemd-networkd' DBus events")

	sigChannel := make(chan *dbus.Signal, 512)
	conn.Signal(sigChannel)

	for v := range sigChannel {
		w := fmt.Sprintf("%v", v.Body[0])

		if strings.HasPrefix(w, networkInterfaceLink) {
			log.Debugf("Received Link DBus signal from 'systemd-networkd'")

			go processDBusLinkMessage(n, v, c)

		} else if strings.HasPrefix(w, "org.freedesktop.network1.Manager") {
			log.Debugf("Received Manager DBus signal from 'systemd-networkd'")

			go processDBusManagerMessage(n, v)
		}
	}

	finished <- true
	return nil
}
