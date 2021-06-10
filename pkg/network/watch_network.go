/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package network

import (
	"strconv"
	"syscall"

	"github.com/network-event-broker/pkg/log"
	"github.com/vishvananda/netlink"
)

func WatchNetwork(n *Network) {
	go n.watchAddresses()
	go n.watchLinks()
}

func (n *Network) watchAddresses() {
	for {
		updates := make(chan netlink.AddrUpdate)
		done := make(chan struct{})

		if err := netlink.AddrSubscribeWithOptions(updates, done, netlink.AddrSubscribeOptions{
			ErrorCallback: func(err error) {
				log.Errorf("Received error from IP address update subscription: %v", err)
			},
		}); err != nil {
			log.Errorf("Failed to subscribe IP address update: %v", err)
		}

		select {
		case <-done:
			log.Infoln("Address watcher failed")
		case updates, ok := <-updates:
			if !ok {
				break
			}

			a := updates.LinkAddress.IP.String()
			mask, _ := updates.LinkAddress.Mask.Size()

			ip := a + "/" + strconv.Itoa(mask)

			log.Infof("Received IP update: %v", updates)

			if updates.NewAddr {
				log.Infof("IP address='%s' added to link ifindex='%d'", ip, updates.LinkIndex)

				n.addOneAddressRule(ip, n.LinksByIndex[updates.LinkIndex], updates.LinkIndex)
			} else {
				log.Infof("IP address='%s' removed from link ifindex='%d'", ip, updates.LinkIndex)

				log.Debugf("Dropping configuration link ifindex='%d' address='%s'", updates.LinkIndex, ip)

				n.dropConfiguration(updates.LinkIndex, ip)
			}
		}
	}
}

func (n *Network) watchLinks() {
	for {
		updates := make(chan netlink.LinkUpdate)
		done := make(chan struct{})

		if err := netlink.LinkSubscribeWithOptions(updates, done, netlink.LinkSubscribeOptions{
			ErrorCallback: func(err error) {
				log.Errorf("Received error from link update subscription: %v", err)
			},
		}); err != nil {
			log.Errorf("Failed to subscribe link update: %v", err)
		}

		select {
		case <-done:
			log.Infoln("Link watcher failed")
		case updates, ok := <-updates:
			if !ok {
				break
			}

			log.Infof("Received Link update: %v", updates)

			n.updateLink(updates)
		}
	}
}

func (n *Network) updateLink(updates netlink.LinkUpdate) {
	n.Mutex.Lock()
	defer n.Mutex.Unlock()

	switch updates.Header.Type {
	case syscall.RTM_DELLINK:

		delete(n.LinksByIndex, int(updates.Index))
		delete(n.LinksByName, updates.Attrs().Name)

	case syscall.RTM_NEWLINK:

		n.LinksByIndex[int(updates.Index)] = updates.Attrs().Name
		n.LinksByName[updates.Attrs().Name] = int(updates.Index)
	}
}

func (n *Network) dropConfiguration(ifIndex int, address string) {
	n.Mutex.Lock()
	defer n.Mutex.Unlock()

	log.Debugf("Dropping routing rules link='%s' ifindex='%d' address='%s'", n.LinksByIndex[ifIndex], ifIndex, address)

	rule, ok := n.RoutingRulesByAddressFrom[address]
	if ok {
		rule.removeRoutingPolicyRule()
		delete(n.RoutingRulesByAddressFrom, address)
	}

	rule, ok = n.RoutingRulesByAddressTo[address]
	if ok {
		rule.removeRoutingPolicyRule()
		delete(n.RoutingRulesByAddressTo, address)
	}

	rt, ok := n.RoutesByIndex[ifIndex]
	if ok {

		if n.isRulesByTableEmpty(rt.Table) {

			log.Debugf("Dropping GW='%s' link='%s' ifindex='%d'  Table='%d'", rt.Gw, n.LinksByIndex[ifIndex], ifIndex, rt.Table)

			rt.removeRoute()
			delete(n.RoutesByIndex, ifIndex)
		}
	}
}
