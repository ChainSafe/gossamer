package main

import (
	"fmt"
	//"fmt"
	//"os"
	//"os/exec"
	//"path"
	"testing"
	"github.com/rendon/testcli"
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


func TestGreetings(t *testing.T) {
	// Using package functions
	testcli.Run("gossamer")
	if !testcli.Success() {
		t.Fatalf("Expected to succeed, but failed: %s", testcli.Error())
	}
	fmt.Println("OUTPUT \n", testcli.Stdout())
	if !testcli.StdoutContains("Hello?") {
		t.Fatalf("Expected %q to contain %q", testcli.Stdout(), "Hello?")
	}
}

func TestGreetingsWithName(t *testing.T) {
	// Using the struct version, if you want to test multiple commands
	c := testcli.Command("gossamer", "--config", "config.toml")
	c.Run()
	if !c.Success() {
		t.Fatalf("Expected to succeed, but failed with error: %s", c.Error())
	}
	fmt.Println("OUTPUT \n", c.Stdout())
	if !c.StdoutContains("Hello John!") {
		t.Fatalf("Expected %q to contain %q", c.Stdout(), "Hello John!")
	}
}

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