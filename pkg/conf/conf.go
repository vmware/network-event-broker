/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package conf

import (
	"os"
	"path"
	"strings"

	"github.com/network-event-broker/pkg/log"
	"github.com/spf13/viper"
)

// App Version
const (
	Version  = "0.1"
	ConfPath = "/etc/network-broker/"
	ConfFile = "network-broker"

	ManagerStateDir = "manager.d"
)

// Config file key value
type Network struct {
	Links string `mapstructure:"Links"`
}
type System struct {
	LogLevel string `mapstructure:"LogLevel"`
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

func Parse() (map[string]int, error) {
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

	if len(c.Network.Links) > 0 {
		log.Infof("Parsed links '%v' from configuration", c.Network.Links)
	}

	links := make(map[string]int)

	s := strings.Split(c.Network.Links, " ")
	for _, c := range s {
		links[c] = 0
	}

	if err := createEventScriptDirs(); err != nil {
		log.Errorf("Failed to create default script state directories: %v", err)
		return nil, err
	}

	return links, nil
}
