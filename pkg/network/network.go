// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 VMware, Inc.

package network

import (
	"errors"
	"strings"
	"sync"

	"github.com/network-event-broker/pkg/conf"
	log "github.com/sirupsen/logrus"
)

type Network struct {
	LinksByName  map[string]int
	LinksByIndex map[int]string

	RoutesByIndex             map[int]*Route
	RoutingRulesByAddressFrom map[string]*RoutingRule
	RoutingRulesByAddressTo   map[string]*RoutingRule

	Mutex *sync.Mutex
}

func New() *Network {
	return &Network{
		LinksByName:  make(map[string]int),
		LinksByIndex: make(map[int]string),

		RoutesByIndex:             make(map[int]*Route),
		RoutingRulesByAddressFrom: make(map[string]*RoutingRule),
		RoutingRulesByAddressTo:   make(map[string]*RoutingRule),
		Mutex:                     &sync.Mutex{},
	}
}

func ConfigureNetwork(link string, n *Network) error {
	n.Mutex.Lock()
	defer n.Mutex.Unlock()

	index, ok := n.LinksByName[link]
	if !ok {
		return errors.New("not found")
	}

	gw, err := GetIpv4Gateway(index)
	if err != nil {
		log.Warnf("Failed to find gateway on link='%s' ifindex='%d' gw='%s'", link, index, err)
		return err
	}

	rt := Route{
		IfIndex: index,
		Gw:      gw,
		Table:   conf.ROUTE_TABLE_BASE + index,
	}

	if err = rt.RouteAdd(); err != nil {
		log.Warnf("Failed to add default gateway on link='%s' ifindex='%d' gw='%s' table='%d: %+v", link, index, gw, rt.Table, err)
		return err
	}

	n.RoutesByIndex[index] = &rt

	log.Debugf("Successfully added default gateway='%s' on link='%s' ifindex='%d' table='%d", gw, link, index, rt.Table)

	existingAddresses, err := getIPv4AddressesByLink(link)
	if err != nil {
		log.Errorf("Failed to fetch Ip addresses of link='%s' ifindex='%d': %+v", link, err)
		return err
	}

	for address := range existingAddresses {
		if err := n.oneAddressRuleAdd(address, link, index); err != nil {
			continue
		}
	}

	return nil
}

func (n *Network) oneAddressRuleAdd(address string, link string, index int) error {
	addr := strings.TrimSuffix(strings.SplitAfter(address, "/")[0], "/")

	from := &RoutingRule{
		From:  addr,
		Table: conf.ROUTE_TABLE_BASE + index,
	}

	if err := from.RoutingPolicyRuleAdd(); err != nil {
		return err
	}

	n.RoutingRulesByAddressFrom[address] = from

	log.Debugf("Successfully added routing policy rule for address='%s' 'from' on link='%s' ifindex='%d' table='%d'", address, link, index, conf.ROUTE_TABLE_BASE+index)

	to := &RoutingRule{
		To:    addr,
		Table: conf.ROUTE_TABLE_BASE + index,
	}

	if err := to.RoutingPolicyRuleAdd(); err != nil {
		return err
	}

	n.RoutingRulesByAddressTo[address] = to

	log.Debugf("Successfully added routing policy rule for address='%s' 'to' on link='%s' ifindex='%d' table='%d", address, link, index, conf.ROUTE_TABLE_BASE+index)

	return nil
}

func (n *Network) isRulesByTableEmpty(table int) bool {
	from := 0
	to := 0

	for _, v := range n.RoutingRulesByAddressFrom {
		if v.Table == table {
			from++
		}
	}

	for _, v := range n.RoutingRulesByAddressTo {
		if v.Table == table {
			to++
		}
	}

	if from == 0 && to == 0 {
		return true
	}

	return false
}
