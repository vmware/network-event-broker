// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 VMware, Inc.

package conf

import (
	"errors"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// App Version
const (
	Version  = "0.1"
	ConfPath = "/etc/network-broker/"
	ConfFile = "network-broker"

	DHClientLeaseFile = "/var/lib/dhclient/dhclient.leases"
	NetworkdLeasePath = "/run/systemd/netif/leases"

	ManagerStateDir = "manager.d"

	ROUTE_TABLE_BASE = 9999

	DefaultLogLevel  = "info"
	DefaultLogFormat = "text"
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
	LogFormat string `mapstructure:"LogFormat"`
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
		os.MkdirAll(path.Join(ConfPath, d), 0755)
	}

	return nil
}

func SetLogLevel(level string) error {
	if level == "" {
		return errors.New("unsupported")
	}

	l, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Warn("Failed to parse log level, falling back to 'info'")
		return errors.New("unsupported")
	} else {
		logrus.SetLevel(l)
	}

	return nil
}

func SetLogFormat(format string) error {
	if format == "" {
		return errors.New("unsupported")
	}

	switch format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			DisableTimestamp: true,
		})

	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
		})

	default:
		logrus.Warn("Failed to parse log format, falling back to 'text'")
		return errors.New("unsupported")
	}

	return nil
}

func Parse() (*Config, error) {
	viper.SetConfigName(ConfFile)
	viper.AddConfigPath(ConfPath)

	if err := viper.ReadInConfig(); err != nil {
		logrus.Errorf("%+v", err)
	}

	viper.SetDefault("System.LogFormat", DefaultLogLevel)
	viper.SetDefault("System.LogLevel", DefaultLogFormat)

	c := Config{}
	if err := viper.Unmarshal(&c); err != nil {
		logrus.Errorf("Failed to parse config file: '/etc/network-broker/network-broker.toml'")
		return nil, err
	}

	if err := SetLogLevel(viper.GetString("NETWORK_EVENT_LOG_LEVEL")); err != nil {
		if err := SetLogLevel(c.System.LogLevel); err != nil {
			c.System.LogLevel = DefaultLogLevel
		}
	}

	logrus.Debugf("Log level set to '%+v'", logrus.GetLevel().String())

	if err := SetLogFormat(viper.GetString("NETWORK_EVENT_LOG_FORMAT")); err != nil {
		if err = SetLogFormat(c.System.LogFormat); err != nil {
			c.System.LogLevel = DefaultLogFormat
		}
	}

	if len(c.System.Generator) > 0 {
		logrus.Infof("Parsed Generator='%v' from configuration", c.System.Generator)
	}
	if len(c.Network.Links) > 0 {
		logrus.Infof("Parsed links='%v' from configuration", c.Network.Links)
	}

	if len(c.Network.RoutingPolicyRules) > 0 {
		logrus.Infof("Parsed RoutingPolicyRules='%+v' from configuration", c.Network.Links)
	}

	if err := createEventScriptDirs(); err != nil {
		logrus.Errorf("Failed to create default script state directories: %+v", err)
		return nil, err
	}

	return &c, nil
}
