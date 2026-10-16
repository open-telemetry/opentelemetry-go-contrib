// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

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

var cgroupContainerIDRe = regexp.MustCompile(`^.*/(?:.*[-:])?([0-9a-f]{64})(?:\.|\s*$)`)

var dockerHostnamePathRe = regexp.MustCompile(`(?:^|/)containers/([0-9a-f]{64})/hostname$`)

// ContainerID returns the container ID from cgroup data or Docker's hostname
// mount. If neither contains a container ID, it returns an empty string.
func ContainerID() (string, error) {
	return containerIDFromFiles(func(path string) (io.ReadCloser, error) {
		return os.Open(path)
	})
}

func containerIDFromFiles(openFile func(string) (io.ReadCloser, error)) (string, error) {
	cgroupFile, cgroupErr := openFile(cgroupPath)
	if cgroupErr == nil {
		id, parseErr := containerIDFromReader(cgroupFile)
		_ = cgroupFile.Close()
		if id != "" {
			return id, nil
		}
		cgroupErr = parseErr
	}

	mountInfoFile, mountInfoErr := openFile(mountInfoPath)
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
	matches := cgroupContainerIDRe.FindStringSubmatch(line)
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

		matches := dockerHostnamePathRe.FindStringSubmatch(fields[3])
		if len(matches) > 1 {
			return matches[1], nil
		}
	}
	return "", scanner.Err()
}
