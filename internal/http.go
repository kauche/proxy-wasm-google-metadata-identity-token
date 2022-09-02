package internal

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

const (
	sharedDataTokenKey         = "proxy-wasm-google-metadata-identity-token-identity-token"
	sharedDataTokenIssuedAtKey = "proxy-wasm-google-metadata-identity-token-identity-token-issued-at"
)

var _ types.HttpContext = (*httpContext)(nil)

type httpContext struct {
	types.DefaultHttpContext

	configuration *pluginConfiguration
}

func (c *httpContext) OnHttpRequestHeaders(_ int, _ bool) types.Action {
	if c.configuration.originalAuthorizationPropagationHeader != "" {
		if err := c.propagateOriginalAuthorizationHeader(); err != nil {
			setErrorHTTPResponseWithLog("failed to propagate the original authorization header: %s", err)
			return types.ActionPause
		}
	}

	tokenBytes, _, tokenErr := proxywasm.GetSharedData(sharedDataTokenKey)
	if tokenErr != nil && tokenErr != types.ErrorStatusNotFound {
		setErrorHTTPResponseWithLog("failed to get the identity token from the shared data: %s", tokenErr)
		return types.ActionPause
	}

	iatBytes, _, iatErr := proxywasm.GetSharedData(sharedDataTokenIssuedAtKey)
	if iatErr != nil && iatErr != types.ErrorStatusNotFound {
		setErrorHTTPResponseWithLog("failed to get the issued at for the identity token from the shared data: %s", iatErr)
		return types.ActionPause
	}

	// If there is an active non-expired cached identity token, use it and do not make a new HTTP call to the metadata server.
	if tokenErr != types.ErrorStatusNotFound && iatErr != types.ErrorStatusNotFound && len(tokenBytes) > 0 && len(iatBytes) > 0 {
		iat := binary.LittleEndian.Uint64(iatBytes)
		now := uint64(time.Now().Unix())

		if now-iat < c.configuration.tokenCacheDuration {
			if err := proxywasm.ReplaceHttpRequestHeader("authorization", fmt.Sprintf("Bearer %s", string(tokenBytes))); err != nil {
				setErrorHTTPResponseWithLog("failed to set the chached identity token to the authorization header: %s", err)
				return types.ActionPause
			}

			return types.ActionContinue
		}
	}

	_, err := proxywasm.DispatchHttpCall(
		c.configuration.metadataServerCluster,
		[][2]string{
			{":path", fmt.Sprintf("/computeMetadata/v1/instance/service-accounts/default/identity?audience=%s", c.configuration.audience)},
			{":method", "GET"},
			{":authority", "metadata.google.internal"},
			{":scheme", "http"},
			{"Metadata-Flavor", "Google"},
		},
		nil,
		nil,
		5000,
		httpCallResponseCallback,
	)
	if err != nil {
		setErrorHTTPResponseWithLog("failed to dispatch the http call for the metadata server: %s", err)
		return types.ActionPause
	}

	return types.ActionPause
}

func (c *httpContext) propagateOriginalAuthorizationHeader() error {
	authorization, err := proxywasm.GetHttpRequestHeader("authorization")
	if err != nil {
		if err == types.ErrorStatusNotFound {
			return nil
		}

		return fmt.Errorf("failed to get the original authorization header: %w", err)
	}

	if err := proxywasm.ReplaceHttpRequestHeader(c.configuration.originalAuthorizationPropagationHeader, authorization); err != nil {
		return fmt.Errorf("failed to propagate the original authorization header as `%s`: %w", c.configuration.originalAuthorizationPropagationHeader, err)
	}

	return nil
}

func httpCallResponseCallback(_, bodySize, _ int) {
	headers, err := proxywasm.GetHttpCallResponseHeaders()
	if err != nil {
		setErrorHTTPResponseWithLog("failed to get the http response headers from the metadata server: %s", err)
		return
	}

	var status string
	for _, header := range headers {
		if header[0] == ":status" {
			status = header[1]
			break
		}
	}

	if status == "" {
		setErrorHTTPResponseWithLog("failed to get the http response status from the metadata server: %s", err)
		return
	}

	res, err := proxywasm.GetHttpCallResponseBody(0, bodySize)
	if err != nil {
		setErrorHTTPResponseWithLog("failed to get the http response from the metadata server: %s", err)
		return
	}

	body := string(res)

	if status != "200" {
		setErrorHTTPResponseWithLog("failed to call the metadata server, status:%s, body:%s", status, body)
		return
	}

	if err := proxywasm.SetSharedData(sharedDataTokenKey, res, 0); err != nil {
		setErrorHTTPResponseWithLog("failed to set the identity token to the shared data: %s", err)
		return
	}

	iatBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(iatBytes, uint64(time.Now().Unix()))
	if err := proxywasm.SetSharedData(sharedDataTokenIssuedAtKey, iatBytes, 0); err != nil {
		setErrorHTTPResponseWithLog("failed to set the issued at for the identity token to the shared data: %s", err)
		return
	}

	if err := proxywasm.ReplaceHttpRequestHeader("authorization", fmt.Sprintf("Bearer %s", body)); err != nil {
		setErrorHTTPResponseWithLog("failed to set the identity token to the authorization header: %s", err)
		return
	}

	proxywasm.ResumeHttpRequest()
}

func setErrorHTTPResponseWithLog(format string, args ...interface{}) {
	proxywasm.LogErrorf(format, args...)
	if err := proxywasm.SendHttpResponse(500, nil, []byte(`{"error": "internal server error"}`), -1); err != nil {
		proxywasm.LogErrorf("failed to set the http error response: %s", err)
	}
}
