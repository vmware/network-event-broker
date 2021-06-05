/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package generators

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/network-event-broker/pkg/bus"
	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/log"
	"github.com/network-event-broker/pkg/network"
	"github.com/network-event-broker/pkg/parser"
	"github.com/network-event-broker/pkg/system"
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

	if err := bus.SetResolve(linkDns, index); err != nil {
		log.Warnln(err)
		return err
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
			log.Errorf("Failed to execute script='%s': %v", script, err)
			continue
		}

		log.Debugf("Successfully executed script='%s' script for link='%s'", script, link)
	}

	return nil
}

func TaskDHClient(n *network.Network, c *conf.Config) error {
	leaseLines, err := system.ReadLines(conf.DHClientLeaseFile)
	if err != nil {
		log.Debugf("Failed to read DHClient lease file '%s': '%v'", conf.DHClientLeaseFile, err)
	}

	if len(leaseLines) <= 0 {
		return errors.New("not found")
	}

	link := "LINK="
	index := "LINKINDEX="
	lease := "DHCP_LEASE="
	idx := 0
	var dnsServers []net.IP

	for _, s := range leaseLines {
		if strings.HasPrefix(s, "lease {") {
			continue
		}

		if strings.HasPrefix(s, "}") {
			executeDHClientLinkStateScripts(n, link, index, lease)
			link = "LINK="
			index = "LINKINDEX="
			lease = "DHCP_LEASE="

			if c.Network.UseDNS {
				setDnsServer(dnsServers, idx)
			}

			continue
		}

		if strings.Contains(s, "interface") {
			i := strings.Index(s, "\"")
			j := strings.LastIndex(s, "\"")

			l := s[i+1 : j]
			link += l

			idx = n.LinksByName[l]
			index += strconv.Itoa(n.LinksByName[link])
			continue
		} else if strings.Contains(s, "domain-name-servers") {
			dns := parser.ParseDNS(s)
			for _, d := range dns {
				v, _ := parser.ParseIP(strings.TrimSpace(d))
				dnsServers = append(dnsServers, v)
			}
		}
		lease += strings.TrimSpace(s)
	}

	return nil
}

func WatchDHClient(n *network.Network, c *conf.Config, finished chan bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("Failed to watch DHClient lease: %+v", err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Debugln(event.Op.String())

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
