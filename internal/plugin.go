package internal

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

var _ types.PluginContext = (*pluginContext)(nil)

type pluginContext struct {
	types.DefaultPluginContext

	configuration *pluginConfiguration
}

func (c *pluginContext) NewHttpContext(_ uint32) types.HttpContext {
	return &httpContext{
		configuration: c.configuration,
	}
}

func (c *pluginContext) OnPluginStart(_ int) types.OnPluginStartStatus {
	config, err := getPluginConfiguration()
	if err != nil {
		proxywasm.LogErrorf("failed to get the plugin configuration: %s", err)
		return types.OnPluginStartStatusFailed
	}

	c.configuration = config

	return types.OnPluginStartStatusOK
}

func getPluginConfiguration() (*pluginConfiguration, error) {
	config, err := proxywasm.GetPluginConfiguration()
	if err != nil {
		if err == types.ErrorStatusNotFound {
			return nil, errors.New("the plugin configuration is not found")
		}

		return nil, fmt.Errorf("failed to get the plugin configuration: %w", err)
	}

	if len(config) == 0 {
		return nil, errors.New("the plugin configuration is empty")
	}

	pc := new(pluginConfiguration)
	if err := json.Unmarshal(config, pc); err != nil {
		return nil, fmt.Errorf("failed to marshal given plugin configuration: %w", err)
	}

	if pc.Audience == "" {
		return nil, fmt.Errorf("the plugin configuration does not include `%s`", configKeyAudience)
	}

	if pc.MetadataServerCluster == "" {
		return nil, fmt.Errorf("the plugin configuration does not include `%s`", configKeyMetadataServerCluster)
	}

	if pc.TokenCacheDuration == 0 {
		pc.TokenCacheDuration = defaultTokenCacheDuration
	}

	return pc, nil
}
