package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	err := _main()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type workflow struct {
	Jobs struct {
		IntegrationTests job `yaml:"integration-tests"`
	} `yaml:"jobs"`
}

type job struct {
	Timeout  int `yaml:"timeout-minutes"`
	Strategy struct {
		Matrix struct {
			Packages []string `yaml:"packages"`
		} `yaml:"matrix"`
	} `yaml:"strategy"`
}

var (
	errPackagesNotEqual = errors.New("packages from workflow and skipped workflow are not equal")
)

func _main() (err error) {
	const moduleName = "github.com/ChainSafe/gossamer"
	rootPath, err := getProjectRootPath(moduleName)
	if err != nil {
		return fmt.Errorf("cannot get project root path: %w", err)
	}

	integrationTestsPath := filepath.Join(rootPath, ".github/workflows/integration-tests.yml")
	integrationTests, err := readAndDecodeWorkflow(integrationTestsPath)
	if err != nil {
		return fmt.Errorf("integration tests workflow: %w", err)
	}

	integrationTestsSkipPath := filepath.Join(rootPath, ".github/workflows/integration-tests-skip.yml")
	integrationTestsSkip, err := readAndDecodeWorkflow(integrationTestsSkipPath)
	if err != nil {
		return fmt.Errorf("integration tests skip workflow: %w", err)
	}

	// Check packages match for normal workflow and skip one
	integrationTestsPackages := integrationTests.Jobs.IntegrationTests.Strategy.Matrix.Packages
	integrationTestsSkipPackages := integrationTestsSkip.Jobs.IntegrationTests.Strategy.Matrix.Packages
	if !stringSlicesAreEqual(integrationTestsPackages, integrationTestsSkipPackages) {
		return fmt.Errorf("%w:\n%s\n%s", errPackagesNotEqual,
			integrationTestsPackages, integrationTestsSkipPackages)
	}

	return nil
}

func readAndDecodeWorkflow(workflowPath string) (workflow workflow, err error) {
	file, err := os.Open(workflowPath) //nolint:gosec
	if err != nil {
		return workflow, fmt.Errorf("cannot open workflow file: %w", err)
	}

	decoder := yaml.NewDecoder(file)

	err = decoder.Decode(&workflow)
	if err != nil {
		_ = file.Close()
		return workflow, fmt.Errorf("cannot decode workflow: %w", err)
	}

	err = file.Close()
	if err != nil {
		return workflow, fmt.Errorf("cannot close workflow file: %w", err)
	}

	return workflow, nil
}

func stringSlicesAreEqual(a, b []string) (equal bool) {
	if len(a) != len(b) {
		return false
	}

	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}

	for _, s := range b {
		_, has := set[s]
		if !has {
			return false
		}
	}

	return true
}

var (
	errFindProjectRoot    = errors.New("cannot find Go project root")
	errModuleNameMismatch = errors.New("module name mismatch")
)

func getProjectRootPath(moduleToFind string) (rootPath string, err error) {
	_, fullpath, _, _ := runtime.Caller(0)
	rootPath = path.Dir(fullpath)

	for {
		pathToCheck := path.Join(rootPath, "go.mod")
		stats, err := os.Stat(pathToCheck)

		if os.IsNotExist(err) {
			previousRootPath := rootPath
			rootPath = path.Dir(rootPath)

			if rootPath == previousRootPath {
				return "", errFindProjectRoot
			}

			continue
		}

		if err != nil {
			return "", err
		}

		if stats.IsDir() {
			continue
		}

		goModFile, err := os.Open(pathToCheck) //nolint:gosec
		if err != nil {
			return "", fmt.Errorf("cannot open go.mod file: %w", err)
		}

		scanner := bufio.NewScanner(goModFile)
		var module string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "module ") {
				module = strings.TrimPrefix(line, "module ")
				break
			}
		}

		if err := scanner.Err(); err != nil {
			_ = goModFile.Close()
			return "", fmt.Errorf("go.mod scanning error: %w", err)
		}

		if module != moduleToFind {
			_ = goModFile.Close()
			return "", fmt.Errorf("%w: looking for %s but found %s",
				errModuleNameMismatch, moduleToFind, module)
		}

		err = goModFile.Close()
		if err != nil {
			return "", fmt.Errorf("cannot close go.mod file: %w", err)
		}

		break
	}

	rootPath, err = filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}

	return rootPath, nil
}
