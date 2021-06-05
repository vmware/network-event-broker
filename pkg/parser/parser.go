// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"errors"
	"net"
	"strings"
)

func ParseIP(ip string) (net.IP, error) {
	if len(ip) == 0 {
		return nil, errors.New("invalid")
	}

	a := net.ParseIP(ip)

	if a.To4() == nil || a.To16() == nil {
		return nil, errors.New("invalid")
	}

	return a, nil
}

func ParseDNS(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "option domain-name-servers ")
	s = strings.TrimSuffix(s, ";")
	s = strings.Replace(s, ",", " ", -1)

	return strings.SplitAfter(s, " ")
}
