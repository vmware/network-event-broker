// SPDX-License-Identifier: Apache-2.0
// Copyright 2023 VMware, Inc.


package network

import (
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func AcquireLinks(n *Network) error {
	linkList, err := netlink.LinkList()
	if err != nil {
		return err
	}

	log.Debugf("Acquiring link information ...")

	for _, link := range linkList {
		if link.Attrs().Name == "lo" {
			continue
		}

		n.LinksByName[link.Attrs().Name] = link.Attrs().Index
		n.LinksByIndex[link.Attrs().Index] = link.Attrs().Name

		log.Debugf("Acquired link='%s' ifindex='%d' from netlink message", link.Attrs().Name, link.Attrs().Index)
	}

	return nil
}
