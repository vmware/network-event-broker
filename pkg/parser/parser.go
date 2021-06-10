// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package parser

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"

	"github.com/network-event-broker/pkg/conf"
)

type Lease struct {
	Interface    string
	Address      string
	ServerName   string
	SubnetMask   string
	Routers      string
	LeaseTime    string
	Server       string
	Hostname     string
	Dns          []string
	DomainSearch []string
	Domain       []string
}

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

func ParseDHClientLease() (map[string]*Lease, error) {
	file, err := os.OpenFile(conf.DHClientLeaseFile, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	leases := make(map[string]*Lease)

	var lease *Lease
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "lease ") && strings.HasSuffix(line, " {") {
			lease = new(Lease)
			continue
		}

		if strings.HasSuffix(line, "}") {
			leases[lease.Interface] = lease
			continue
		}

		switch {
		case strings.Contains(line, "interface"):
			lease.Interface = strings.Split(line, "\"")[1]
		case strings.Contains(line, "fixed-address"):
			lease.Address = strings.TrimSuffix(strings.Split(line, " ")[1], ";")
		case strings.Contains(line, "subnet-mask"):
			lease.SubnetMask = strings.TrimSuffix(strings.Split(line, " ")[2], ";")
		case strings.Contains(line, "routers"):
			lease.Routers = strings.TrimSuffix(strings.Split(line, " ")[2], ";")
		case strings.Contains(line, "dhcp-server-identifier"):
			lease.Server = strings.TrimSuffix(strings.Split(line, " ")[2], ";")
		case strings.Contains(line, "domain-name-servers"):
			lease.Dns = strings.Split(strings.TrimSuffix(strings.Split(line, " ")[2], ";"), ",")
		case strings.Contains(line, "domain-name"):
			s := strings.TrimSuffix(strings.ReplaceAll(line, "option domain-name", ""), ";")
			s = strings.ReplaceAll(s, ",", "")
			t := strings.Split(s, "\"")

			for _, d := range t {
				if strings.TrimSpace(d) == "" {
					continue
				}
				lease.Domain = append(lease.Domain, d)
			}
		case strings.Contains(line, "host-name"):
			lease.Hostname = strings.Split(line, "\"")[1]
		case strings.Contains(line, "domain-search"):
			s := strings.TrimSuffix(strings.ReplaceAll(line, "option domain-search", ""), ";")
			s = strings.ReplaceAll(s, ",", "")
			lease.DomainSearch = strings.Split(s, "\"")
		}
	}

	return leases, nil
}
