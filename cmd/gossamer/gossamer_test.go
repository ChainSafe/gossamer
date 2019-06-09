package main_test

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/rendon/testcli"
	"go/build"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"
)

var binaryname = "gossamer-test"

const configTest = "config-test.toml"
const timeFormat     = "2006-01-02T15:04:05-0700"

func setup() {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	r := exec.Command("go", "build", "-o", gopath + "/bin/gossamer-test")
	err := r.Run()
	if err != nil {
		log15.Crit("could not execute binary", "executable",binaryname, "err",err)
		os.Exit(1)
	}
	run := exec.Command(`gossamer-test`)
	err = run.Run()
	if err != nil {
		log15.Crit("could not execute binary", "executable",binaryname, "err",err)
		os.Exit(1)
	}
	fp, err := os.Create(configTest)
	if err != nil {
		log15.Crit("could not create test config", "config",configTest, "err",err)
		os.Exit(1)
	}

	testConfig := fmt.Sprintf("%s%s%s%v%s%s%v%s%s%s",
		"[ServiceConfig]\n", "BootstrapNodes = []\n", "Port = ",
		7005,"\n","RandSeed = ", 0, "\n\n", "[DbConfig]\n",
		"Datadir = \"\"\x0A")

	_, err = fp.WriteString(testConfig)
	if err != nil {
		log15.Crit("could not write to test config", "config","config-test.toml", "err",err)
		os.Exit(1)
	}
}

func teardown() {
	err := os.Chdir("../gossamer")
	if err != nil {
		log15.Error("could not change dir", "err", err)
		os.Exit(1)
	}
	if err := os.RemoveAll("./chaindata"); err != nil {
		log15.Warn("removal of temp directory bin failed", "err",err)
	}
	if err := os.RemoveAll("./config-test.toml"); err != nil {
		log15.Warn("removal of temp config.toml failed", "err",err)
	}
}

func TestInitialOutput(t *testing.T) {
	setup()
	testcli.Run("gossamer-test")
	if !testcli.Success() {
		teardown()
		t.Fatalf("Expected to succeed, but failed: %s", testcli.Error())
	}
	output := fmt.Sprintf("%s%v%s", "t=", time.Now().Format(timeFormat), " lvl=info msg=\"🕸️ starting p2p service\" blockchain=gossamer\x0A")
	if !testcli.StdoutContains(output) {
		teardown()
		t.Fatalf("Expected %q to contain %q", testcli.Stdout(), output)
	}
	if !reflect.DeepEqual(testcli.Stdout(), output) {
		teardown()
		t.Fatalf("actual = %s, expected = %s", testcli.Stdout(), output)
	}
	defer teardown()
}

func TestCliArgs(t *testing.T) {
	setup()
	res := expectedResponses()
	defer teardown()
	tests := []struct {
		name    string
		args    []string
		expected string
	}{
		{"dumpconfig with config set", []string{"--config", "config-test.toml","dumpconfig"}, res[0]},
		{"config specified", []string{"--config", "config-test.toml"}, res[1]},
		{"default config", []string{}, res[1]},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				gopath = build.Default.GOPATH
			}
			dir := gopath + "/bin/"
			cmd := exec.Command(path.Join(dir, binaryname), tt.args...)
			actual, err := cmd.CombinedOutput()
			if err != nil {
				teardown()
				t.Fatal(err)
			}
			if i > 0 {
				if !strings.ContainsAny(string(actual), res[1]) {
					teardown()
					t.Fatalf("actual = %s, expected = %s", string(actual), res[1])
				}
			} else {
			if !reflect.DeepEqual(string(actual), res[0]) {
				teardown()
				t.Fatalf("actual = %s, expected = %s", actual, res[0])
			}}
		})
	}
}

func expectedResponses() []string {
	dumpConfigExp := fmt.Sprintf("%s%s%s%v%s%s%v%s%s%s",
		"[ServiceConfig]\n", "BootstrapNodes = []\n", "Port = ",
		7005,"\n","RandSeed = ", 0, "\n\n", "[DbConfig]\n",
		"Datadir = \"\"\x0A")

	startingChainMsg := fmt.Sprintf("%s%v%s", "t=", time.Now().Format(timeFormat),
		" lvl=info msg=\"🕸️ starting p2p service\" blockchain=gossamer\x0A")

	response := []string{dumpConfigExp, startingChainMsg}

	return response
}