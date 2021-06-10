/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package generators

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/network-event-broker/pkg/bus"
	"github.com/network-event-broker/pkg/conf"
	"github.com/network-event-broker/pkg/log"
	"github.com/network-event-broker/pkg/network"
	"github.com/network-event-broker/pkg/system"
)

const (
	networkInterface  = "org.freedesktop.network1"
	networkObjectPath = "/org/freedesktop/network1"

	networkInterfaceLink       = "org.freedesktop.network1.Link"
	networkInterfaceLinkEscape = networkInterfaceLink + "/_3"
)

func executeNetworkdLinkStateScripts(link string, index int, k string, v string) error {
	scriptDirs, err := system.ReadAllScriptDirs(conf.ConfPath)
	if err != nil {
		log.Errorf("Failed to find any scripts in conf dir: %+v", err)
		return err
	}

	for _, d := range scriptDirs {
		stateDir := strings.Trim(v, "\"") + ".d"

		if stateDir == d {
			scripts, err := system.ReadAllScriptInConfDir(path.Join(conf.ConfPath, d))
			if err != nil {
				log.Errorf("Failed to read script dir '%s'", path.Join(conf.ConfPath, d))
				continue
			}

			path.Join(conf.ConfPath, d)
			linkNameEnvArg := "LINK=" + link
			linkIndexEnvArg := "LINKINDEX=" + strconv.Itoa(index)
			linkStateEnvArg := k + "=" + strings.Trim(v, "\"")

			if len(scripts) <= 0 {
				continue
			}

			leaseFile := path.Join(conf.NetworkdLeasePath, strconv.Itoa(index))
			leaseLines, err := system.ReadLines(leaseFile)
			if err != nil {
				log.Debugf("Failed to read lease file of link='%+v'", link, err)
			}

			var leaseArg string
			if len(leaseLines) > 0 {
				leaseArg = "DHCP_LEASE="
				leaseArg += strings.Join(leaseLines, " ")
			}

			for _, s := range scripts {
				script := path.Join(conf.ConfPath, d, s)

				log.Debugf("Executing script '%s' in dir='%v' for link='%s'", script, d, link)

				cmd := exec.Command(script)
				cmd.Env = append(os.Environ(),
					linkNameEnvArg,
					linkNameEnvArg,
					linkIndexEnvArg,
					linkStateEnvArg,
					leaseArg,
				)

				if err := cmd.Run(); err != nil {
					log.Errorf("Failed to execute script='%s': %v", script, err)
					continue
				}

				log.Debugf("Successfully executed script '%s' in dir='%v' for link='%s'", script, d, link)
			}
		}
	}

	return nil
}

func executeNetworkdManagerScripts(k string, v string) error {
	managerStatePath := path.Join(conf.ConfPath, conf.ManagerStateDir)

	scripts, err := system.ReadAllScriptInConfDir(managerStatePath)
	if err != nil {
		log.Errorf("Failed to read script dir '%s'", managerStatePath)
		return nil
	}

	for _, s := range scripts {
		script := path.Join(managerStatePath, s)

		log.Debugf("Executing script '%s' in dir='%s'", script, managerStatePath)

		managerStateEnvArg := k + "=" + v
		cmd := exec.Command(script)
		cmd.Env = append(os.Environ(),
			managerStateEnvArg,
			managerStateEnvArg,
		)

		if err := cmd.Run(); err != nil {
			log.Errorf("Failed to execute script='%s': %+v", script, err)
			continue
		}

		log.Debugf("Successfully executed script '%s' in dir='%v' for manager state", script, managerStatePath)
	}

	return nil
}

func processDBusLinkMessage(n *network.Network, v *dbus.Signal, c *conf.Config) error {
	if !strings.HasPrefix(string(v.Path), networkInterfaceLinkEscape) {
		return nil
	}

	strIndex := strings.TrimPrefix(string(v.Path), networkInterfaceLinkEscape)
	index, err := strconv.Atoi(strIndex)
	if err != nil {
		log.Errorf("Failed to convert ifindex=\"%s\" to integer: %+v", strIndex, err)
		return nil
	}

	log.Debugf("Received DBus signal from systemd-networkd for ifindex='%d' link='%s'", index, n.LinksByIndex[index])

	linkState := v.Body[1].(map[string]dbus.Variant)
	for k, v := range linkState {
		switch k {
		case "OperationalState":
			{
				log.Debugf("Link='%s' ifindex='%d' changed state '%s'=%s", n.LinksByIndex[index], index, k, v)

				if c.Network.Links != "" {
					if strings.Contains(c.Network.Links, n.LinksByIndex[index]) {
						executeNetworkdLinkStateScripts(n.LinksByIndex[index], index, k, v.String())
					}
				} else {
					executeNetworkdLinkStateScripts(n.LinksByIndex[index], index, k, v.String())
				}

				if strings.Trim(v.String(), "\"") == "routable" && strings.Contains(c.Network.RoutingPolicyRules, n.LinksByIndex[index]) {
					network.ConfigureNetwork(n.LinksByIndex[index], n)
				}
			}
		}
	}

	return nil
}

func processDBusManagerMessage(n *network.Network, v *dbus.Signal) error {
	state := v.Body[1].(map[string]dbus.Variant)

	for k, v := range state {
		log.Debugf("Manager chaged state '%v='%v'", k, v.String())

		executeNetworkdManagerScripts(k, v.String())
	}

	return nil
}

func WatchNetworkd(n *network.Network, c *conf.Config, finished chan bool) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatalf("Failed to connect to system bus: %v", err)
		os.Exit(1)
	}
	defer conn.Close()

	opts := []dbus.MatchOption{
		dbus.WithMatchSender(networkInterface),
		dbus.WithMatchInterface(bus.DBusProperties),
		dbus.WithMatchMember("PropertiesChanged"),
	}

	if err := conn.AddMatchSignal(opts...); err != nil {
		log.Errorf("Failed to add match signal for '%s`: %+v", networkInterface, err)
		return err
	}

	log.Infoln("Listening to systemd-networkd DBus events")

	sigChannel := make(chan *dbus.Signal, 512)
	conn.Signal(sigChannel)

	for v := range sigChannel {
		w := fmt.Sprintf("%v", v.Body[0])

		if strings.HasPrefix(w, networkInterfaceLink) {
			log.Debugf("Received Link DBus signal from systemd-networkd'")

			go processDBusLinkMessage(n, v, c)

		} else if strings.HasPrefix(w, "org.freedesktop.network1.Manager") {
			log.Debugf("Received Manager DBus signal from systemd-networkd'")

			go processDBusManagerMessage(n, v)
		}
	}

	finished <- true

	return nil
}
