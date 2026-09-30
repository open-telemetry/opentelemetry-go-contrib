// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
)

const cgroupPath = "/proc/self/cgroup"

var cgroupContainerIDRe = regexp.MustCompile(`^.*/(?:.*[-:])?([0-9a-f]{64})(?:\.|\s*$)`)

var (
	defaultOSStat = os.Stat
	osStat        = defaultOSStat

	defaultOSOpen = func(name string) (io.ReadCloser, error) {
		return os.Open(name)
	}
	osOpen = defaultOSOpen
)

// ContainerID returns the container ID from the cgroup file.
// If the cgroup file does not exist or contains no container ID, it returns
// an empty string and no error.
func ContainerID() (string, error) {
	if _, err := osStat(cgroupPath); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	file, err := osOpen(cgroupPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	return containerIDFromReader(file)
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
