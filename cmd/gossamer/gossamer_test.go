package main

import (
	"fmt"
	"github.com/ChainSafe/gossamer/internal/cmdtest"
	"github.com/rendon/testcli"
	"io/ioutil"
	"reflect"

	"time"

	//"reflect"
	//"time"
	//
	//"github.com/rendon/testcli"
	//"fmt"
	//"os"
	//"os/exec"
	//"path"
	"testing"
)

var binaryname = "gossamer"

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

const timeFormat     = "2006-01-02T15:04:05-0700"

func tmpdir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "gossamer-test")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

type testgeth struct {
	*cmdtest.TestCmd

	// template variables for expect
	Datadir   string
}

func runGeth(t *testing.T, args ...string) *testgeth {
	tt := &testgeth{}
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, arg := range args {
			fmt.Println(i, arg)

		}

	//tt.Datadir = tmpdir(t)
	//fmt.Println(tt.Datadir)
	//tt.Cleanup = func() { os.RemoveAll(tt.Datadir) }
	//args = append([]string{"--datadir", tt.Datadir}, args...)
	//// Remove the temporary datadir if something fails below.
	//defer func() {
	//	if t.Failed() {
	//		tt.Cleanup()
	//	}
	//}()


	// Boot "geth". This actually runs the test binary but the TestMain
	// function will prevent any tests from running.
	tt.Run("gossamer", args...)

	return tt
}

func TestWelcome(t *testing.T) {
	g := runGeth(t, "--config config.toml")
	g.SetTemplateFunc("ti", func() string { return time.Now().Format(timeFormat) })
	g.Expect(`t=, {{ti}}, " lvl=info msg="🕸️ starting p2p service" blockchain=gossamer`)

	g.ExpectExit()
}


func TestGreetings(t *testing.T) {
	// Using package functions
	testcli.Run("gossamer")
	if !testcli.Success() {
		t.Fatalf("Expected to succeed, but failed: %s", testcli.Error())
	}
	fmt.Println("OUTPUT \n", testcli.Stdout())
	output := fmt.Sprintf("%s%v%s", "t=", time.Now().Format(timeFormat), " lvl=info msg=\"🕸️ starting p2p service\" blockchain=gossamer")
	//if !testcli.StdoutContains(output) {
	//	//	t.Fatalf("Expected %q to contain %q", testcli.Stdout(), output)
	//	//}
	fmt.Println([]byte(")"))
	b := []byte(output)
	a := []byte(testcli.Stdout())
	fmt.Println("exp", b)
	fmt.Println("act", a)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("actual = %s, expected = %s", testcli.Stdout(), output)
	}
}

//func TestGreetingsWithName(t *testing.T) {
//	// Using the struct version, if you want to test multiple commands
//	c := testcli.Command("gossamer", "--config", "config.toml")
//	c.Run()
//	if !c.Success() {
//		t.Fatalf("Expected to succeed, but failed with error: %s", c.Error())
//	}
//	fmt.Println("OUTPUT \n", c.Stdout())
//	if !c.StdoutContains("Hello John!") {
//		t.Fatalf("Expected %q to contain %q", c.Stdout(), "Hello John!")
//	}
//}

//func TestMain(m *testing.M) {
//	err := os.Chdir("..")
//	if err != nil {
//		fmt.Printf("could not change dir: %v", err)
//		os.Exit(1)
//	}
//	make := exec.Command("make gossamer")
//	err = make.Run()
//	if err != nil {
//		fmt.Printf("could not make binary for %s: %v", binaryname, err)
//		os.Exit(1)
//	}
//
//	os.Exit(m.Run())
//}