// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

const cgroupPath = "/proc/self/cgroup"

const mountInfoPath = "/proc/self/mountinfo"

var cgroupContainerIDRegEx = regexp.MustCompile(`^.*/(?:.*[-:])?([0-9a-f]{64})(?:\.|\s*$)`)

var dockerHostnamePathRegEx = regexp.MustCompile(`(?:^|/)containers/([0-9a-f]{64})/hostname$`)

func (d *dockerProviderImpl) containerIDFromFiles() (string, error) {
	cgroupFile, cgroupErr := d.openFile(cgroupPath)
	if cgroupErr == nil {
		id, parseErr := containerIDFromReader(cgroupFile)
		_ = cgroupFile.Close()
		if id != "" {
			return id, nil
		}
		cgroupErr = parseErr
	}

	mountInfoFile, mountInfoErr := d.openFile(mountInfoPath)
	if mountInfoErr == nil {
		id, parseErr := containerIDFromMountInfo(mountInfoFile)
		_ = mountInfoFile.Close()
		if id != "" {
			return id, nil
		}
		mountInfoErr = parseErr
	}

	if errors.Is(cgroupErr, os.ErrNotExist) {
		cgroupErr = nil
	}
	if errors.Is(mountInfoErr, os.ErrNotExist) {
		mountInfoErr = nil
	}
	return "", errors.Join(cgroupErr, mountInfoErr)
}

func containerIDFromReader(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if id := containerIDFromLine(scanner.Text()); id != "" {
			return id, nil
		}
	}
	return "", scanner.Err()
}

func containerIDFromLine(line string) string {
	matches := cgroupContainerIDRegEx.FindStringSubmatch(line)
	if len(matches) <= 1 {
		return ""
	}
	return matches[1]
}

func containerIDFromMountInfo(reader io.Reader) (string, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 || fields[4] != "/etc/hostname" {
			continue
		}

		matches := dockerHostnamePathRegEx.FindStringSubmatch(fields[3])
		if len(matches) > 1 {
			return matches[1], nil
		}
	}
	return "", scanner.Err()
}
