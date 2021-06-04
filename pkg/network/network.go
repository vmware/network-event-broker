/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package network

import (
	"errors"
	"strconv"
	"strings"
	"syscall"

	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/log"
	"github.com/vishvananda/netlink"
)

type Network struct {
	LinksByName  map[string]int
	LinksByIndex map[int]string

	RoutingRulesByAddressFrom map[string]*RoutingRule
	RoutingRulesByAddressTo   map[string]*RoutingRule
}

func ConfigureNetwork(link string, n *Network) error {
	index, ok := n.LinksByName[link]
	if !ok {
		return errors.New("not found")
	}

	existingAddresses, err := GetIPv4AddressesByLink(link)
	if err != nil {
		log.Errorf("Failed to fetch Ip addresses of link='%s' ifindex='%d': %+v", link, err)
		return err
	}

	gw, err := GetIpv4Gateway(index)
	if err != nil {
		return err
	}

	if err = AddRoute(index, conf.ROUTE_TABLE_BASE+index, gw); err != nil {
		log.Warnf("Failed to add default gateway on link='%s' ifindex='%d' gw='%s': %+v", link, index, gw, err)
		return err
	}

	log.Debugf("Successfully added default gateway='%s' on link='%s' ifindex='%d'", gw, link, index)

	for address := range existingAddresses {
		addr := strings.TrimSuffix(strings.SplitAfter(address, "/")[0], "/")

		from := &RoutingRule{
			From:  addr,
			Table: ROUTE_TABLE_BASE + index,
		}

		if err := AddRoutingPolicyRule(from); err != nil {
			return err
		}

		n.RoutingRulesByAddressFrom[address] = from

		log.Debugf("Successfully added routing policy rule 'from' on link='%s' ifindex='%d'", link, index)

		to := &RoutingRule{
			To:    addr,
			Table: ROUTE_TABLE_BASE + index,
		}

		if err := AddRoutingPolicyRule(to); err != nil {
			return err
		}

		n.RoutingRulesByAddressTo[address] = to

		log.Debugf("Successfully added routing policy rule 'to' on link='%s' ifindex='%d'", link, index)
	}

	return nil
}

func WatchAddresses(n *Network) {
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
			} else {
				log.Infof("IP address='%s' removed from link ifindex='%d'", ip, updates.LinkIndex)

				log.Debugf("Dropping configuration link ifindex='%d' address='%s'", updates.LinkIndex, ip)

				DropConfiguration(n, updates.LinkIndex, ip)
			}
		}
	}
}

func DropConfiguration(n *Network, ifIndex int, address string) {
	log.Debugf("Dropping routing rules link='%s' ifindex='%d' address='%s'", n.LinksByIndex[ifIndex], ifIndex, address)

	rule, ok := n.RoutingRulesByAddressFrom[address]
	if ok {
		RemoveRoutingPolicyRule(rule)
	}

	delete(n.RoutingRulesByAddressFrom, address)

	rule, ok = n.RoutingRulesByAddressTo[address]
	if ok {
		RemoveRoutingPolicyRule(rule)
	}

	delete(n.RoutingRulesByAddressTo, address)
}

func WatchLinks(n *Network) {
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

			if updates.Header.Type == syscall.RTM_DELLINK {
				link := n.LinksByIndex[int(updates.Index)]

				delete(n.LinksByIndex, int(updates.Index))
				delete(n.LinksByName, link)
			} else if updates.Header.Type == syscall.RTM_NEWLINK {
				n.LinksByIndex[int(updates.Index)] = updates.Attrs().Name
				n.LinksByName[updates.Attrs().Name] = int(updates.Index)
			}
		}
	}
}
