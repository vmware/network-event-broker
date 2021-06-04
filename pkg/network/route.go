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

func GetDefaultIpv4Gateway() (string, error) {
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

func GetDefaultIpv4GatewayByLink(ifIndex int) (string, error) {
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

func GetIpv4GatewayByLink(ifIndex int) (string, error) {
	routes, err := netlink.RouteList(nil, syscall.AF_INET)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.LinkIndex == ifIndex {
			if route.Gw != nil && route.Gw.To4().String() != "" {
				return route.Gw.To4().String(), nil
			}
		}
	}

	return "", errors.New("not found")
}

func GetIpv4Gateway(ifIndex int) (string, error) {
	gw, err := GetDefaultIpv4GatewayByLink(ifIndex)
	if err != nil {
		gw, err = GetIpv4GatewayByLink(ifIndex)
		if err != nil {
			// Try Harder ?
			gw, err = GetDefaultIpv4Gateway()
			if err != nil {
				return "", err
			}
		}
	}

	return gw, nil
}

func AddRoute(route *Route) error {
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

func RemoveRoute(route *Route) error {
	rt := netlink.Route{
		LinkIndex: route.IfIndex,
		Gw:        net.ParseIP(route.Gw).To4(),
		Table:     route.Table,
	}

	if err := netlink.RouteDel(&rt); err != nil && err.Error() != "file exists" {
		return err
	}

	return nil
}
