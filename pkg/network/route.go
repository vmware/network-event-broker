// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"errors"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

type Route struct {
	Table   int
	IfIndex int
	Gw      string
}

func getDefaultIpv4Gateway() (string, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
			if route.Gw.To4() == nil {
				return "", errors.New("failed to find gateway, default route is present")
			}

			return route.Gw.To4().String(), nil
		}
	}

	return "", errors.New("not found")
}

func getDefaultIpv4GatewayByLink(ifIndex int) (string, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.Dst == nil || route.Dst.String() == "0.0.0.0/0" {
			if route.LinkIndex == ifIndex {
				return route.Gw.To4().String(), nil
			}
		}
	}

	return "", errors.New("not found")
}

func getIpv4GatewayByLink(ifIndex int) (string, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.LinkIndex == ifIndex {
			if route.Dst != nil && route.Dst.String() != "0.0.0.0/0" {
				return route.Dst.String(), nil
			}
		}
	}

	return "", errors.New("not found")
}

func getIpv4Gateway(ifIndex int) (string, error) {
	gw, err := getDefaultIpv4GatewayByLink(ifIndex)
	if err != nil {
		gw, err = getIpv4GatewayByLink(ifIndex)
		if err != nil {
			gw, err = getDefaultIpv4Gateway()
			if err != nil {
				return "", err
			}
		}
	}

	return gw, nil
}

func (route *Route) addRoute() error {
	rt := netlink.Route{
		LinkIndex: route.IfIndex,
		Gw:        net.ParseIP(route.Gw).To4(),
		Table:     route.Table,
	}

	if err := netlink.RouteAdd(&rt); err != nil && err.Error() != "file exists" {
		return err
	}

	return nil
}

func (route *Route) removeRoute() error {
	rt := netlink.Route{
		LinkIndex: route.IfIndex,
		Gw:        net.ParseIP(route.Gw).To4(),
		Table:     route.Table,
	}

	if err := netlink.RouteDel(&rt); err != nil {
		return err
	}

	return nil
}
