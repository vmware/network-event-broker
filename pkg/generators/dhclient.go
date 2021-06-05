/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package generators

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/log"
	"github.com/network-event-broker/pkg/network"
	"github.com/network-event-broker/pkg/system"
)

func executeDHClientLinkStateScripts(n *network.Network, link string, lease string) error {
	scripts, err := system.ReadAllScriptInConfDir(path.Join(conf.ConfPath, "routable.d"))
	if err != nil {
		log.Errorf("Failed to read script dir: %+v", err)
		return err
	}

	for _, s := range scripts {
		script := path.Join(conf.ConfPath, "routable.d", s)

		log.Debugf("Executing script='%s' for link='%s' lease=%s", script, link, lease)

		cmd := exec.Command(script)
		cmd.Env = append(os.Environ(),
			link,
			link,
			lease,
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

func TaskDHClient(n *network.Network) error {
	leaseLines, err := system.ReadLines(conf.DHClientLeaseFile)
	if err != nil {
		log.Debugf("Failed to read dhclient lease file '%s': '%v'", conf.DHClientLeaseFile, err)
	}

	if len(leaseLines) <= 0 {
		return errors.New("not found")
	}

	linkNameEnvArg := "LINK="
	leaseArg := "DHCP_LEASE="

	for _, s := range leaseLines {
		if strings.HasPrefix(s, "lease {") {
			continue
		}

		if strings.HasPrefix(s, "}") {
			executeDHClientLinkStateScripts(n, linkNameEnvArg, leaseArg)
			linkNameEnvArg = "LINK="
			leaseArg = "DHCP_LEASE="
			continue
		}

		if strings.Contains(s, "interface") {
			i := strings.Index(s, "\"")
			j := strings.LastIndex(s, "\"")

			linkNameEnvArg += s[i+1 : j-1]
			continue
		}
		leaseArg += strings.TrimSpace(s)
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

				TaskDHClient(n)

			case err := <-watcher.Errors:
				log.Errorln(err)
			}
		}
	}()

	if err := watcher.Add(conf.DHClientLeaseFile); err != nil {
		fmt.Println("ERROR", err)
	}

	<-done
}
