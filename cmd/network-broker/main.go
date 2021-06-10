/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package main

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/listeners"
	"github.com/network-event-broker/pkg/log"
	"github.com/network-event-broker/pkg/network"
)

func main() {
	log.Init()

	c, err := conf.Parse()
	if err != nil {
		log.Warnf("Failed to parse configuration: %v", err)
	}

	n := network.New()
	if n == nil {
		log.Fatalln("Failed to create network. Aborting ...")
		os.Exit(1)
	}

	err = network.AcquireLinks(n)
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
