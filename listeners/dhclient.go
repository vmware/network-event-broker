// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 VMware, Inc.


package listeners

import (
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/vmware/network-event-broker/pkg/bus"
	"github.com/vmware/network-event-broker/pkg/conf"
	"github.com/vmware/network-event-broker/pkg/network"
	"github.com/vmware/network-event-broker/pkg/parser"
	"github.com/vmware/network-event-broker/pkg/system"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func setDnsServer(dnsServers []net.IP, index int) error {
	linkDns := make([]bus.DnsServer, len(dnsServers))
	for i, s := range dnsServers {
		linkDns[i] = bus.DnsServer{
			Family:  unix.AF_INET,
			Address: []byte(s.To4()),
		}
	}

	if err := bus.SetResolveDNS(linkDns, index); err != nil {
		log.Warnln(err)
	}

	return nil
}

func setDnsDomain(dnsDomains []string, index int) error {
	linkDomains := make([]bus.Domain, len(dnsDomains))
	for i, domain := range dnsDomains {
		linkDomains[i] = bus.Domain{
			Domain: domain,
			Set:    true,
		}
	}

	if err := bus.SetResolveDomain(linkDomains, index); err != nil {
		log.Warnln(err)
	}

	return nil
}

func executeDHClientLinkStateScripts(n *network.Network, link string, strIndex string, lease string) error {
	scripts, err := system.ReadAllScriptInConfDir(path.Join(conf.ConfPath, "routable.d"))
	if err != nil {
		log.Errorf("Failed to read script dir: %+v", err)
		return err
	}

	for _, s := range scripts {
		script := path.Join(conf.ConfPath, "routable.d", s)

		log.Debugf("Executing script='%s' for '%s' lease=%s", script, link, lease)

		cmd := exec.Command(script)
		cmd.Env = append(os.Environ(),
			link,
			link,
			strIndex,
			lease,
		)

		if err := cmd.Run(); err != nil {
			log.Errorf("Failed to execute script='%s': %w", script, err)
			continue
		}

		log.Debugf("Successfully executed script='%s' for link='%s'", script, link)
	}

	return nil
}

func TaskDHClient(n *network.Network, c *conf.Config) error {
	leases, err := parser.ParseDHClientLease()
	if err != nil {
		log.Debugf("Failed to parse DHClient lease file '%s': %w", conf.DHClientLeaseFile, err)
	}

	for i, lease := range leases {
		_, ok := n.LinksByName[i]
		if !ok {
			continue
		}

		idx := n.LinksByName[i]
		if c.Network.Links != "" {
			if !strings.Contains(c.Network.Links, n.LinksByIndex[idx]) {
				return nil
			}
		}

		strIndex := strconv.Itoa(idx)

		link := "LINK=" + i
		index := "LINKINDEX=" + strIndex
		dns := strings.Join(lease.Dns, ",")
		domain := strings.Join(lease.Domain, ",")
		strings.Join(lease.Domain, ",")
		dhcpLease := "DHCP_LEASE=" + "ADDRESS=" + lease.Address + ",DNS=" + strings.Join(lease.Dns, ",") + ",ROUTER=" + lease.Routers + ",SUBNETMASK=" + lease.SubnetMask + ",DNS=" + dns + ",DOMAIN=" + domain

		executeDHClientLinkStateScripts(n, link, index, dhcpLease)

		if c.Network.UseHostname {
			if err := bus.SetHostname(lease.Hostname); err != nil {
				log.Warnln("Failed to set hostname='%s': %+w", lease.Hostname, err)
			}
		}

		if c.Network.UseDNS && len(lease.Dns) > 0 {
			var dnsServers []net.IP

			for _, d := range lease.Dns {
				v, _ := parser.ParseIP(strings.TrimSpace(d))
				dnsServers = append(dnsServers, v)
			}
			setDnsServer(dnsServers, idx)
		}

		if c.Network.UseDomain && len(lease.Domain) > 0 {
			setDnsDomain(lease.Domain, idx)
		}
	}

	return nil
}

func WatchDHClient(n *network.Network, c *conf.Config, finished chan bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Failed to watch DHClient lease: %w", err)
	}
	defer watcher.Close()

	log.Infoln("Listening to DHClient events")

	// Try once incase dhclient already have the leases
	TaskDHClient(n, c)

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Debugf("DHClient Received event: %s", event.Op.String())

				TaskDHClient(n, c)

			case err := <-watcher.Errors:
				log.Errorln(err)
			}
		}
	}()

	if err := watcher.Add(conf.DHClientLeaseFile); err != nil {
		log.Errorf("Failed to watch DHClient lease file: %w", err)
	}

	<-done
}
