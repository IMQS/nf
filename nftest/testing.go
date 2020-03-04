package nftest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/IMQS/nf/nfdb"
)

// MakeDBConfig returns a DBConfig that is configured to run against the
// docker Postgres image imqs/postgres:unittest-10.5 (or any other version).
// If eraseDB is true, then wipe all tables in this database before returning
func MakeDBConfig(dbname string, eraseDB bool) nfdb.DBConfig {
	cfg := nfdb.DBConfig{
		Driver:   "postgres",
		Database: "unittest_" + dbname,
		Host:     "localhost",
		Username: "unittest_user",
		Password: "unittest_password",
	}
	if eraseDB {
		if err := nfdb.DropAllTables(nil, cfg.Driver, cfg.DSN()); err != nil {
			fmt.Printf("Warning: DropAllTables failed: %v\n", err)
		}
	}
	return cfg
}

// PingUntil200 repeatedly tries to contact pingURL until it receives a 200 response code.
// If, after timeout, we have still not received a 200, then we call t.Fatal()
func PingUntil200(t *testing.T, timeout time.Duration, pingURL string) {
	// once the server is ping-able, we can continue
	start := time.Now()
	pingOK := false
	sleepMS := 1
	t.Logf("Pinging %v until service comes alive", pingURL)
	for time.Now().Sub(start) < time.Second {
		resp, err := http.DefaultClient.Get(pingURL)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err == nil && resp.StatusCode == 200 {
			pingOK = true
			break
		}
		time.Sleep(time.Duration(sleepMS) * time.Millisecond)
		sleepMS *= 2
	}
	if !pingOK {
		t.Fatalf("Failed to ping server at %v after initial startup", pingURL)
	}
}

func turnAnyIntoJSON(t *testing.T, any interface{}) []byte {
	if s, ok := any.(string); ok {
		// assume requestBody is already JSON, since it's a string
		return []byte(s)
	}

	// encode requestBody to JSON
	j, err := json.Marshal(any)
	if err != nil {
		t.Fatalf("Failed to marshal 'any' into JSON: %v", err)
	}
	return j
}

func assertJSONEquals(t *testing.T, url string, expect []byte, actual []byte) {
	// I don't know how better to normalize JSON in Go. A brief google search didn't yield anything simple
	expectComp := bytes.Buffer{}
	actualComp := bytes.Buffer{}
	err1 := json.Compact(&expectComp, expect)
	err2 := json.Compact(&actualComp, actual)
	if err1 != nil {
		t.Fatalf("compacting error (1): %v", err1)
	}
	if err2 != nil {
		t.Fatalf("compacting error (2): %v", err2)
	}
	if actualComp.String() != expectComp.String() {
		t.Errorf("Response from %v failed JSON comparison\nExpected:\n%v\nActual:\n%v\n", url, string(expect), string(actual))
	}
}

// ValidateResponse validates the response body of 'resp', ensuring that it meets the criteria of expectResponse.
//
// The validation depends on expectResponse:
//  expectResponse  Validation
//  int             Ensure that the HTTP status code is equal to expectResponse
//  >>foo           Ensure that the string 'foo' can be found in the response body
//  <other>         Marshal expectResponse to JSON, and ensure the JSON matches the response body exactly (with whitespace removed)
func ValidateResponse(t *testing.T, resp *http.Response, url string, expectResponse interface{}) *[]byte {
	var err error
	defer resp.Body.Close()
	respBody := []byte{}
	if resp != nil {
		respBody, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body from %v: %v", url, err)
		}
	}

	if expectStatus, ok := expectResponse.(int); ok {
		if resp.StatusCode != expectStatus {
			t.Errorf("Response from %v: expected to receive %v, but got %v\n%v", url, expectStatus, resp.StatusCode, string(respBody))
		}
	} else {
		if str, ok := expectResponse.(string); ok {
			if strings.HasPrefix(str, ">>") {
				// ensure we CAN find 'seek'
				seek := str[2:]
				if strings.Index(string(respBody), seek) == -1 {
					t.Errorf("Response from %v: expected to find %v, but got\n%v\n", url, seek, string(respBody))
				}
				return &respBody
			} else if strings.HasPrefix(str, "!>>") {
				// ensure we can't find 'seek'
				seek := str[2:]
				if strings.Index(string(respBody), seek) != -1 {
					t.Errorf("Response from %v: expected to NOT find %v, but got\n%v\n", url, seek, string(respBody))
				}
				return &respBody
			}
		}
		assertJSONEquals(t, url, turnAnyIntoJSON(t, expectResponse), respBody)
	}

	return &respBody
}

// POSTJson sends a JSON object to the server, and calls ValidateResponse on the result.
func POSTJson(t *testing.T, url string, requestBody interface{}, expectResponse interface{}) *[]byte {
	resp, err := http.DefaultClient.Post(url, "application/json", bytes.NewReader(turnAnyIntoJSON(t, requestBody)))
	if err != nil {
		t.Fatalf("Failed to connect to %v: %v", url, err)
	}
	return ValidateResponse(t, resp, url, expectResponse)
}

// GETAndValidateJson hits the given URL, and calls ValidateResponse on the result.
func GETAndValidateJson(t *testing.T, url string, expectResponse interface{}) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to connect to %v: %v", url, err)
	}
	ValidateResponse(t, resp, url, expectResponse)
}

// GETJson returns a byte array that responsd t
func GETBytes(t *testing.T, url string) ([]byte, error) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to connect to %v: %v", url, err)
		return nil, err
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Response failed with the following status code: %v", resp.StatusCode)
		return nil, err
	}
	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read content body")
		return nil, err
	}
	return out, nil
}

// GETDump gets the given URL and writes it to the test log.
func GETDump(t *testing.T, url string) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		t.Fatalf("Failed to connect to %v: %v", url, err)
	}
	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body from %v: %v", url, err)
	}
	t.Logf("%v:\n%v", url, string(r))
}
