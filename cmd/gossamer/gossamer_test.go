package main

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/rendon/testcli"
	"os"
	"os/exec"
	"reflect"

	"time"
	"testing"
)

var binaryname = "gossamer-test"

const timeFormat     = "2006-01-02T15:04:05-0700"

func TestMain(m *testing.M) {
	r := exec.Command("go", "build", "-o", "./bin/gossamer-test")
	err := r.Run()
	if err != nil {
		log15.Crit("could not make binary", "executable",binaryname, "err",err)
		os.Exit(1)
	}
	err = os.Chdir("./bin")
	if err != nil {
		log15.Crit("could not change dir", "err",err)
		os.Exit(1)
	}
	run := exec.Command(`./gossamer-test`)
	err = run.Run()
	if err != nil {
		log15.Crit("could not make binary", "executable",binaryname, "err",err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestInitialOutput(t *testing.T) {
	testcli.Run("./gossamer-test")
	if !testcli.Success() {
		t.Fatalf("Expected to succeed, but failed: %s", testcli.Error())
	}
	output := fmt.Sprintf("%s%v%s", "t=", time.Now().Format(timeFormat), " lvl=info msg=\"🕸️ starting p2p service\" blockchain=gossamer\x0A")
	if !testcli.StdoutContains(output) {
		t.Fatalf("Expected %q to contain %q", testcli.Stdout(), output)
	}
	if !reflect.DeepEqual(testcli.Stdout(), output) {
		t.Fatalf("actual = %s, expected = %s", testcli.Stdout(), output)
	}
	defer func() {
		err := os.Chdir("..")
		if err != nil {
			log15.Error("could not change dir", "err", err)
			os.Exit(1)
		}
		if err := os.RemoveAll("./bin"); err != nil {
			log15.Warn("removal of temp directory bin failed", "err",err)
		}
	}()
}

//func TestGreetingsWithName(t *testing.T) {
//	// Using the struct version, if you want to test multiple commands
//	c := testcli.Command("gossamer", "--config", "config.toml")
//	c.Run()
//	if !c.Success() {
//		t.Fatalf("Expected to succeed, but failed with error: %s", c.Error())
//	}
//	fmt.Println("OUTPUT \n", c.Stdout())
//	if !c.StdoutContains("TBD") {
//		t.Fatalf("Expected %q to contain %q", c.Stdout(), "Hello John!")
//	}
//}

//func TestCliArgs(t *testing.T) {
//	tests := []struct {
//		name    string
//		args    []string
//		fixture string
//	}{
//		{"no arguments", []string{}, "no-args.golden"},
//		{"one argument", []string{"ciao"}, "one-argument.golden"},
//		{"multiple arguments", []string{"ciao", "hello"}, "multiple-arguments.golden"},
//		{"shout arg", []string{"--shout", "ciao"}, "shout-arg.golden"},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			dir, err := os.Getwd()
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			cmd := exec.Command(path.Join(dir, binaryname), tt.args...)
//			output, err := cmd.CombinedOutput()
//			if err != nil {
//				t.Fatal(err)
//			}
//			fmt.Println(output)
//			//if !reflect.DeepEqual(actual, expected) {
//			//	t.Fatalf("actual = %s, expected = %s", actual, expected)
//			//}
//		})
//	}
//}

