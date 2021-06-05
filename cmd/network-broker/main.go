/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/generators"
	"github.com/network-event-broker/pkg/log"
	"github.com/network-event-broker/pkg/network"
)

func main() {
	log.Init()

	c, err := conf.Parse()
	if err != nil {
		log.Warnf("Failed to parse configuration: %v", err)
	}

	/* Refresh link information */
	n, err := network.AcquireLinks()
	if err != nil {
		log.Fatalf("Failed to acquire link information. Unable to continue: %v", err)
		os.Exit(1)
	}

	// Watch network
	go network.WatchAddresses(n)
	go network.WatchLinks(n)

	finished := make(chan bool)

	go generators.WatchNetworkdDBusEvents(n, c, finished)
	go generators.WatchDHClient(n, c, finished)

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		os.Exit(0)
	}()

	<-finished
}
