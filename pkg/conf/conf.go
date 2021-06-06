/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package conf

import (
	"os"
	"path"

	"github.com/network-event-broker/pkg/log"
	"github.com/spf13/viper"
)

// App Version
const (
	Version           = "0.1"
	ConfPath          = "/etc/network-broker/"
	ConfFile          = "network-broker"
	DHClientLeaseFile = "/var/lib/dhclient/dhclient.leases"

	ManagerStateDir = "manager.d"

	ROUTE_TABLE_BASE = 9999
)

// Config file key value
type Network struct {
	Links              string `mapstructure:"Links"`
	RoutingPolicyRules string `mapstructure:"RoutingPolicyRules"`
	UseDNS             bool   `mapstructure:"UseDNS"`
	UseDomain          bool   `mapstructure:"UseDomain"`
	UseHostname        bool   `mapstructure:"UseHostname"`
}

type System struct {
	Generator string `mapstructure:"Generator"`
	LogLevel  string `mapstructure:"LogLevel"`
}
type Config struct {
	Network Network `mapstructure:"Network"`
	System  System  `mapstructure:"System"`
}

func createEventScriptDirs() error {
	var linkEventStateDirs [6]string

	linkEventStateDirs[0] = "no-carrier.d"
	linkEventStateDirs[1] = "carrier.d"
	linkEventStateDirs[2] = "degraded.d"
	linkEventStateDirs[3] = "routable.d"
	linkEventStateDirs[4] = "configured.d"
	linkEventStateDirs[5] = ManagerStateDir

	for _, d := range linkEventStateDirs {
		os.MkdirAll(path.Join(ConfPath, d), 07777)
	}

	return nil
}

func Parse() (*Config, error) {
	viper.SetConfigName(ConfFile)
	viper.AddConfigPath(ConfPath)

	if err := viper.ReadInConfig(); err != nil {
		log.Errorf("%+v", err)
	}

	c := Config{}
	if err := viper.Unmarshal(&c); err != nil {
		log.Errorf("Failed to parse config file: '/etc/network-broker/network-broker.toml'")
		return nil, err
	}

	log.SetLevel(c.System.LogLevel)

	if len(c.System.Generator) > 0 {
		log.Infof("Parsed Generator='%v' from configuration", c.System.Generator)
	}
	if len(c.Network.Links) > 0 {
		log.Infof("Parsed links='%v' from configuration", c.Network.Links)
	}

	if len(c.Network.RoutingPolicyRules) > 0 {
		log.Infof("Parsed RoutingPolicyRules='%v' from configuration", c.Network.Links)
	}

	if err := createEventScriptDirs(); err != nil {
		log.Errorf("Failed to create default script state directories: %v", err)
		return nil, err
	}

	return &c, nil
}
