// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"os"
	"os/exec"
	"path"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/vmware/network-event-broker/pkg/conf"
)

func ExecuteScripts(link string, index int) error {
	scripts, err := ReadAllScriptInConfDir(path.Join(conf.ConfPath, conf.RoutesModifiedDir))
	if err != nil {
		log.Errorf("Failed to read script dir '%s'", path.Join(conf.ConfPath, conf.RoutesModifiedDir))
		return err
	}

	if len(scripts) <= 0 {
		log.Debugf("No script in '%+v'", conf.RoutesModifiedDir)
		return err
	}

	path.Join(conf.ConfPath, conf.RoutesModifiedDir)
	linkNameEnvArg := "LINK=" + link
	linkIndexEnvArg := "LINKINDEX=" + strconv.Itoa(index)

	for _, s := range scripts {
		script := path.Join(conf.ConfPath, conf.RoutesModifiedDir, s)

		log.Debugf("Executing script '%s' in dir='%v' for link='%s'", script, conf.RoutesModifiedDir, link)

		cmd := exec.Command(script)
		cmd.Env = append(os.Environ(),
			linkNameEnvArg,
			linkNameEnvArg,
			linkIndexEnvArg,
		)

		if err := cmd.Run(); err != nil {
			log.Errorf("Failed to execute script='%s': %v", script, err)
			continue
		}

		log.Debugf("Successfully executed script '%s' in dir='%v' for link='%s'", script, conf.RoutesModifiedDir, link)
	}

	return nil
}
