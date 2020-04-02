package rpc

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestHttpServer(t *testing.T) {
	cfg := &HTTPServerConfig{
		Host: "localhost",
		Port: 8545,
	}

	server := NewHTTPServer(cfg)
	err := server.Start()
	if err != nil {
		t.Fatal(err)
	}

	defer server.Stop()

	client := &http.Client{}

	data := []byte(`{"jsonrpc":"2.0","method":"non_existent","params":[],"id":1}`)
	buf := &bytes.Buffer{}
	_, err = buf.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8545", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	t.Log(resp)

	body, _ := ioutil.ReadAll(resp.Body)
	t.Log(body)
}

func TestHttpServer_valid(t *testing.T) {
	cfg := &HTTPServerConfig{
		Host: "localhost",
		Port: 8546,
	}

	server := NewHTTPServer(cfg)
	err := server.Start()
	if err != nil {
		t.Fatal(err)
	}

	defer server.Stop()

	client := &http.Client{}

	data := []byte(`{"jsonrpc":"2.0","method":"system_NetworkState","params":[],"id":1}`)
	buf := &bytes.Buffer{}
	_, err = buf.Write(data)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:8545", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	t.Log(resp)

	body, _ := ioutil.ReadAll(resp.Body)
	t.Log(body)
}
