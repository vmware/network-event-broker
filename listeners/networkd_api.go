// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 VMware, Inc.

package listeners

import (
	"strconv"
	"strings"

	"github.com/network-event-broker/pkg/configfile"
)

func ParseLinkString(ifindex int, key string) (string, error) {
	path := "/run/systemd/netif/links/" + strconv.Itoa(ifindex)
	v, err := configfile.ParseKeyFromSectionString(path, "", key)
	if err != nil {
		return "", err
	}

	return v, nil
}

func ParseLinkSetupState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "ADMIN_STATE")
}

func ParseLinkCarrierState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "CARRIER_STATE")
}

func ParseLinkOnlineState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "ONLINE_STATE")
}

func ParseLinkActivationPolicy(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "ACTIVATION_POLICY")
}

func ParseLinkNetworkFile(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "NETWORK_FILE")
}

func ParseLinkOperationalState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "OPER_STATE")
}

func ParseLinkAddressState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "ADDRESS_STATE")
}

func ParseLinkIPv4AddressState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "IPV4_ADDRESS_STATE")
}

func ParseLinkIPv6AddressState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "IPV6_ADDRESS_STATE")
}

func ParseLinkDNS(ifindex int) ([]string, error) {
	s, err := ParseLinkString(ifindex, "DNS")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}

func ParseLinkNTP(ifindex int) ([]string, error) {
	s, err := ParseLinkString(ifindex, "NTP")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}

func ParseLinkDomains(ifindex int) ([]string, error) {
	s, err := ParseLinkString(ifindex, "DOMAINS")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}

func ParseNetworkState(key string) (string, error) {
	v, err := configfile.ParseKeyFromSectionString("/run/systemd/netif/state", "", key)
	if err != nil {
		return "", err
	}

	return v, nil
}

func ParseNetworkOperationalState() (string, error) {
	return ParseNetworkState("OPER_STATE")
}

func ParseNetworkCarrierState() (string, error) {
	return ParseNetworkState("CARRIER_STATE")
}

func ParseNetworkAddressState() (string, error) {
	return ParseNetworkState("ADDRESS_STATE")
}

func ParseNetworkIPv4AddressState() (string, error) {
	return ParseNetworkState("IPV4_ADDRESS_STATE")
}

func ParseNetworkIPv6AddressState() (string, error) {
	return ParseNetworkState("IPV6_ADDRESS_STATE")
}

func ParseNetworkOnlineState() (string, error) {
	return ParseNetworkState("ONLINE_STATE")
}

func ParseNetworkDNS() ([]string, error) {
	s, err := ParseNetworkState("DNS")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}

func ParseNetworkNTP() ([]string, error) {
	s, err := ParseNetworkState("NTP")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}

func ParseNetworkDomains() ([]string, error) {
	s, err := ParseNetworkState("DOMAINS")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}

func ParseNetworkRouteDomains() ([]string, error) {
	s, err := ParseNetworkState("ROUTE_DOMAINS")
	if err != nil {
		return nil, err
	}

	return strings.Split(s, " "), nil
}
