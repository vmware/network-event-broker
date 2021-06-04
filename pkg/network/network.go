/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package network

import (
	"errors"
	"strings"

	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/log"
)

func ConfigureNetwork(ifname string) error {
	n, err := AcquireLinks()
	if err != nil {
		log.Errorf("Failed to fetch links: %+v", err)
		return err
	}

	index, ok := n.LinksByName[ifname]
	if !ok {
		return errors.New("not found")
	}

	existingAddresses, err := GetIPv4Addresses(ifname)
	if err != nil {
		log.Errorf("Failed to fetch Ip addresses of link='%s' ifindex='%d': %+v", ifname, err)
		return err
	}

	for a := range existingAddresses {

		gw, err := GetIpv4Gateway(index)
		if err != nil {
			continue
		}

		if err = AddRoute(index, conf.ROUTE_TABLE_BASE+index, gw); err != nil {
			log.Warnf("Failed to add route on link='%s' ifindex='%d' gw='%s': %+v", ifname, index, gw, err)
		}

		a := strings.TrimSuffix(strings.SplitAfter(a, "/")[0], "/")

		from := &IPRoutingRule{
			From:  a,
			Table: ROUTE_TABLE_BASE + index,
		}

		if err := AddRoutingPolicyRule(from); err != nil {
			return err
		}

		to := &IPRoutingRule{
			To:    a,
			Table: ROUTE_TABLE_BASE + index,
		}

		if err := AddRoutingPolicyRule(to); err != nil {
			return err
		}
	}

	return nil
}
