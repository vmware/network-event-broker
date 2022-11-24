// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 VMware, Inc.

package main

import (
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/vmware/network-event-broker/listeners"
	"github.com/vmware/network-event-broker/pkg/conf"
	"github.com/vmware/network-event-broker/pkg/network"
	"github.com/vmware/network-event-broker/pkg/system"
	log "github.com/sirupsen/logrus"
)

func run(c *conf.Config) {
	n := network.New()
	if n == nil {
		log.Fatalln("Failed to create network. Aborting ...")
		os.Exit(1)
	}

	err := network.AcquireLinks(n)
	if err != nil {
		log.Fatalf("Failed to acquire link information. Unable to continue: %v", err)
		os.Exit(1)
	}

	// Watch network
	go network.WatchNetwork(n)

	finished := make(chan bool)

	if c.System.Generator == "" || strings.Contains(c.System.Generator, "systemd-networkd") {
		log.Infoln("Starting listener: 'systemd-netword")
		go listeners.WatchNetworkd(n, c, finished)
	} else {
		log.Infoln("Starting listener: 'dhclient'")
		go listeners.WatchDHClient(n, c, finished)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		os.Exit(0)
	}()

	<-finished
}

func main() {
	c, err := conf.Parse()
	if err != nil {
		log.Warnf("Failed to parse configuration: %v", err)
	}

	log.Infof("network-broker: v%s (built %s)", conf.Version, runtime.Version())

	cred, err := system.GetUserCredentials("")
	if err != nil {
		log.Warningf("Failed to get current user credentials: %+v", err)
		os.Exit(1)
	} else {
		if cred.Uid == 0 {
			u, err := system.GetUserCredentials("network-broker")
			if err != nil {
				log.Errorf("Failed to get user 'network-broker' credentials: %+v", err)
				os.Exit(1)
			} else {
				if err := system.EnableKeepCapability(); err != nil {
					log.Warningf("Failed to enable keep capabilities: %+v", err)
				}

				if err := system.SwitchUser(u); err != nil {
					log.Warningf("Failed to switch user: %+v", err)
				}

				if err := system.DisableKeepCapability(); err != nil {
					log.Warningf("Failed to disable keep capabilities: %+v", err)
				}

				err := system.ApplyCapability(u)
				if err != nil {
					log.Warningf("Failed to apply capabilities: +%v", err)
				}
			}
		}
	}

	run(c)
}
