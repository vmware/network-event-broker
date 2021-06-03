/* SPDX-License-Identifier: Apache-2.0
 * Copyright © 2021 VMware, Inc.
 */

package system

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// PathExists test if path exists
func PathExists(path string) bool {
	_, r := os.Stat(path)
	if os.IsNotExist(r) {
		return false
	}

	return true
}

func ReadAllScriptDirs(path string) ([]string, error) {
	var confDirs []string

	if !PathExists(path) {
		return nil, errors.New("Failed to open script conf dir '/etc/networkd-afterburn'")
	}

	dirs, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}

		if filepath.Ext(d.Name()) == ".d" {
			confDirs = append(confDirs, d.Name())
		}
	}

	return confDirs, nil
}

func ReadAllScriptInConfDir(dir string) ([]string, error) {
	var scripts []string

	if !PathExists(dir) {
		return nil, errors.New("Failed to open script conf dir in '/etc/networkd-afterburn'")
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		scripts = append(scripts, f.Name())
	}

	return scripts, nil
}

func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "#") {
			continue
		}

		lines = append(lines, scanner.Text())

	}

	return lines, scanner.Err()
}
