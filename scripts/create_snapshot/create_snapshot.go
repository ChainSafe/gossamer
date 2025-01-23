// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const checkInterval = 2 * time.Second

func main() {
	threshold, dbLocation, snapshotDestination := parseArgs(os.Args)

	for {
		containerID, err := findContainerID("gossamer")
		if err != nil {
			fmt.Println(err)
			time.Sleep(checkInterval)
			continue
		}

		blockHeight, err := fetchMetric("gossamer_network_syncer_blocks_synced_total")
		if err != nil {
			fmt.Println(err)
			time.Sleep(checkInterval)
			continue
		}
		fmt.Println("Snapshotting at block height:", blockHeight)

		if blockHeight >= threshold {

			if err := stopContainer(containerID); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Stopped gossamer container (ID %s)\n", containerID)

			dstDir := filepath.Join(snapshotDestination, fmt.Sprintf("block-%d", blockHeight), "db")
			fmt.Printf("Copying DB to %s ...\n", dstDir)
			if err := copyDirectory(dbLocation, dstDir); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if err := startContainer(containerID); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Started gossamer container (ID %s)\n", containerID)
			break
		}

		time.Sleep(checkInterval)
	}
}

func findContainerID(name string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("creating Docker client: %v", err)
	}

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("fetching containers: %v", err)
	}

	for _, c := range containers {
		for _, containerName := range c.Names {
			if strings.Contains(containerName, name) {
				return c.ID, nil
			}
		}
	}

	return "", errors.New("container not found")
}

func fetchMetric(metricName string) (uint32, error) {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := httpClient.Get("http://localhost:9876/metrics")
	if err != nil {
		return 0, fmt.Errorf("fetching '%s' metric: %w", metricName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to fetch metric '%s'. status code: %d", metricName, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading metrics response: %w", err)
	}

	metricsText := string(body)
	valueStr := ""
	scanner := bufio.NewScanner(strings.NewReader(metricsText))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, metricName) {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				valueStr = parts[1]
				break
			}
		}
	}

	if valueStr == "" {
		return 0, fmt.Errorf("metric '%s' not found", metricName)
	}

	blockHeight, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid metric value: %s", valueStr)
	}

	return uint32(blockHeight), nil
}

func stopContainer(containerID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("creating Docker client: %w", err)
	}

	timeout := int(10 * time.Second)
	if err := cli.ContainerStop(context.Background(), containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("stopping container: %w", err)
	}
	return nil
}

func startContainer(containerID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("creating Docker client: %w", err)
	}

	if err := cli.ContainerStart(context.Background(), containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("starting container: %w", err)
	}
	return nil
}

func copyDirectory(srcDir, dstDir string) error {
	srcDir = filepath.Clean(srcDir)
	dstDir = filepath.Clean(dstDir)

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func copyFile(srcFile, dstFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	srcInfo, err := src.Stat()
	if err != nil {
		return err
	}
	return os.Chmod(dstFile, srcInfo.Mode())
}

func parseArgs(args []string) (threshold uint32, dbLocation string, snapshotDestination string) {
	if len(args) < 4 {
		fmt.Printf("Usage: %s <approximate block height> <db location> <snapshot destination>\n", path.Base(args[0]))
		os.Exit(1)
	}

	thresholdStr := args[1]
	t, err := strconv.ParseUint(thresholdStr, 10, 32)
	if err != nil {
		fmt.Println("Invalid threshold value.")
		os.Exit(1)
	}
	threshold = uint32(t)

	dbLocation = args[2]
	snapshotDestination = args[3]

	if !isDirectory(dbLocation) {
		fmt.Printf("DB location is not a directory or not readable: %s\n", dbLocation)
		os.Exit(1)
	}

	if !isWritableDirectory(snapshotDestination) {
		fmt.Printf("Snapshot destination is not a writable directory: %s\n", snapshotDestination)
		os.Exit(1)
	}

	return
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isWritableDirectory(path string) bool {
	file, err := os.CreateTemp(path, "test")
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(file.Name())
	return true
}
