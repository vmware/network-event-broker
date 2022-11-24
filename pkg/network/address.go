// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"github.com/vishvananda/netlink"
)

func getIPv4AddressesByLink(name string) (map[string]bool, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	addresses, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	m := make(map[string]bool)
	for _, addr := range addresses {
		m[addr.IPNet.String()] = true
	}

	return m, nil
}
