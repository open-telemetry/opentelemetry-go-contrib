package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const expectedGoVersion = "go 1.23.0"

func main() {
	if err := verifyGoVersion(); err != nil {
		log.Fatal(err)
	}
}

func verifyGoVersion() error {
	var modFiles []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Base(path) == "go.mod" {
			modFiles = append(modFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil
	}
	if len(modFiles) == 0 {
		return fmt.Errorf("the ")
	}

	for _, file := range modFiles {
		bytes, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		content := string(bytes)
		if strings.Contains(content, "toolchain") {
			return fmt.Errorf("xxx")
		}

		contents := strings.Split(content, "\n")
		goVersionFound := false

		for _, line := range contents {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "go ") {
				goVersionFound = true
				if line != expectedGoVersion {
					return fmt.Errorf("xxxx")
				}
				break
			}
		}

		if !goVersionFound {
			return fmt.Errorf("xxxx")
		}
	}

	return nil
}
