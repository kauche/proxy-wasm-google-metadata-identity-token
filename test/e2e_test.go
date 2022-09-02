package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	echoserverpb "github.com/110y/echoserver/echoserver/api/v1"
)

func TestE2E(t *testing.T) {
	t.Parallel()

	t.Run("with token_cache_duration and original_authorization_propagation_header", func(t *testing.T) {
		t.Parallel()

		host := "upstream-1"

		req1, err := createHTTPRequest(host)
		if err != nil {
			t.Errorf("failed to create a http request: %s", err)
			return
		}

		req1.Header.Set("authorization", "original-authorization")

		res1, err := http.DefaultClient.Do(req1)
		if err != nil {
			t.Errorf("failed to send first http request: %s", err)
			return
		}
		defer res1.Body.Close()

		if res1.StatusCode != 200 {
			t.Errorf("invalid http status code for the first http request: %s", res1.Status)
			return
		}

		echores1 := new(echoserverpb.EchoResponse)
		if err := json.NewDecoder(res1.Body).Decode(echores1); err != nil {
			t.Errorf("failed to marshal the first response to json: %s", err)
			return
		}

		authorization1, ok := echores1.Headers["authorization"]
		if !ok {
			t.Error("authorization header for the first http request is not found")
			return
		}
		if len(authorization1.Value) != 1 {
			t.Errorf("authorization header for the first http request includes invalid number of values: %d", len(authorization1.Value))
			return
		}
		if !strings.HasPrefix(authorization1.Value[0], "Bearer identity-token-for-upstream-1-") {
			t.Errorf("invalid authorization header for the first http request: %s", authorization1.Value[0])
			return
		}

		originalAuthorization, ok := echores1.Headers["original-authorization"]
		if !ok {
			t.Error("original-authorization header is not found")
			return
		}

		if len(originalAuthorization.Value) != 1 {
			t.Errorf("original-authorization header for the first http request includes invalid number of values: %d", len(originalAuthorization.Value))
			return
		}

		if originalAuthorization.Value[0] != "original-authorization" {
			t.Errorf("invalid original-authorization header: %s", originalAuthorization.Value[0])
			return
		}

		req2, err := createHTTPRequest(host)
		if err != nil {
			t.Errorf("failed to create a http request: %s", err)
			return
		}

		res2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Errorf("failed to send second http request: %s", err)
			return
		}

		if res2.StatusCode != 200 {
			t.Errorf("invalid http status code for the second http request: %s", res1.Status)
			return
		}
		defer res2.Body.Close()

		echores2 := new(echoserverpb.EchoResponse)
		if err := json.NewDecoder(res2.Body).Decode(echores2); err != nil {
			t.Errorf("failed to marshal the second response to json: %s", err)
			return
		}

		authorization2, ok := echores2.Headers["authorization"]
		if !ok {
			t.Error("authorization header for the second http request is not found")
			return
		}
		if len(authorization2.Value) != 1 {
			t.Errorf("authorization header for the second http request includes invalid number of values: %d", len(authorization1.Value))
			return
		}

		if authorization1.Value[0] != authorization2.Value[0] {
			t.Errorf("authorization headers for the first and second http request are not same, 1:%s, 2:%s", authorization1.Value[0], authorization2.Value[0])
			return
		}

		// wait for the cached identity token to be expired
		time.Sleep(5 * time.Second)

		req3, err := createHTTPRequest(host)
		if err != nil {
			t.Errorf("failed to create a http request: %s", err)
			return
		}

		res3, err := http.DefaultClient.Do(req3)
		if err != nil {
			t.Errorf("failed to send third http request: %s", err)
			return
		}

		if res3.StatusCode != 200 {
			t.Errorf("invalid http status code for the third http request: %s", res1.Status)
			return
		}
		defer res3.Body.Close()

		echores3 := new(echoserverpb.EchoResponse)
		if err := json.NewDecoder(res3.Body).Decode(echores3); err != nil {
			t.Errorf("failed to marshal the third response to json: %s", err)
			return
		}

		authorization3, ok := echores3.Headers["authorization"]
		if !ok {
			t.Error("authorization header for the third http request is not found")
			return
		}
		if len(authorization3.Value) != 1 {
			t.Errorf("authorization header for the third http request includes invalid number of values: %d", len(authorization1.Value))
			return
		}
		if !strings.HasPrefix(authorization3.Value[0], "Bearer identity-token-for-upstream-1-") {
			t.Errorf("invalid authorization header for the third http request: %s", authorization1.Value[0])
			return
		}

		if authorization1.Value[0] == authorization3.Value[0] {
			t.Errorf("the cached identity token is not expired: %s", authorization3.Value[0])
			return
		}
	})

	t.Run("without token_cache_duration and original_authorization_propagation_header", func(t *testing.T) {
		t.Parallel()

		host := "upstream-2"

		req1, err := createHTTPRequest(host)
		if err != nil {
			t.Errorf("failed to create a http request: %s", err)
			return
		}

		res1, err := http.DefaultClient.Do(req1)
		if err != nil {
			t.Errorf("failed to send first http request: %s", err)
			return
		}
		defer res1.Body.Close()

		if res1.StatusCode != 200 {
			t.Errorf("invalid http status code for the first http request: %s", res1.Status)
			return
		}

		echores1 := new(echoserverpb.EchoResponse)
		if err := json.NewDecoder(res1.Body).Decode(echores1); err != nil {
			t.Errorf("failed to marshal the first response to json: %s", err)
			return
		}

		authorization1, ok := echores1.Headers["authorization"]
		if !ok {
			t.Error("authorization header for the first http request is not found")
			return
		}
		if len(authorization1.Value) != 1 {
			t.Errorf("authorization header for the first http request includes invalid number of values: %d", len(authorization1.Value))
			return
		}
		if !strings.HasPrefix(authorization1.Value[0], "Bearer identity-token-for-upstream-2-") {
			t.Errorf("invalid authorization header for the first http request: %s", authorization1.Value[0])
			return
		}

		// wait for the cached identity token to be expired
		time.Sleep(5 * time.Second)

		req2, err := createHTTPRequest(host)
		if err != nil {
			t.Errorf("failed to create a http request: %s", err)
			return
		}

		res2, err := http.DefaultClient.Do(req2)
		if err != nil {
			t.Errorf("failed to send second http request: %s", err)
			return
		}

		if res2.StatusCode != 200 {
			t.Errorf("invalid http status code for the second http request: %s", res1.Status)
			return
		}
		defer res2.Body.Close()

		echores2 := new(echoserverpb.EchoResponse)
		if err := json.NewDecoder(res2.Body).Decode(echores2); err != nil {
			t.Errorf("failed to marshal the second response to json: %s", err)
			return
		}

		authorization2, ok := echores2.Headers["authorization"]
		if !ok {
			t.Error("authorization header for the second http request is not found")
			return
		}
		if len(authorization2.Value) != 1 {
			t.Errorf("authorization header for the second http request includes invalid number of values: %d", len(authorization1.Value))
			return
		}

		if authorization1.Value[0] != authorization2.Value[0] {
			t.Errorf("authorization headers for the first and second http request are not same, 1:%s, 2:%s", authorization1.Value[0], authorization2.Value[0])
			return
		}
	})

	t.Run("with an invalid metadataserver cluster", func(t *testing.T) {
		t.Parallel()
		host := "upstream-3"

		req, err := createHTTPRequest(host)
		if err != nil {
			t.Errorf("failed to create a http request: %s", err)
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("failed to send first http request: %s", err)
			return
		}
		defer res.Body.Close()

		if res.StatusCode != 500 {
			t.Errorf("expected an internal server error, but got statusl:%d", res.StatusCode)
			return
		}
	})
}

func createHTTPRequest(host string) (*http.Request, error) {
	addr := os.Getenv("ENVOY_ADDRESS")
	if addr == "" {
		addr = "localhost:9090"
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/", addr), bytes.NewBuffer([]byte(`{"message":"hello"}`)))
	if err != nil {
		return nil, err
	}

	req.Host = host
	req.Header.Set("content-type", "application/json")

	return req, nil
}
