package rpc

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestHttpServer(t *testing.T) {
	t.Skip()
	client := &http.Client{}

	data := []byte(`{"jsonrpc":"2.0","method":"system_Health","params":[],"id":1}`)
	buf := &bytes.Buffer{}
	_, err := buf.Write(data)
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
