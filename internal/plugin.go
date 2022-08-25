package internal

import (
	"errors"
	"fmt"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
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

	if !gjson.ValidBytes(config) {
		return nil, errors.New("the plugin configuration is not valid JSON")
	}

	jsonConfig := gjson.ParseBytes(config)

	audience := jsonConfig.Get(configKeyAudience).String()
	if audience == "" {
		return nil, fmt.Errorf("the plugin configuration does not include `%s`", configKeyAudience)
	}

	cluster := jsonConfig.Get(configKeyMetadataServerCluster).String()
	if cluster == "" {
		return nil, fmt.Errorf("the plugin configuration does not include `%s`", configKeyMetadataServerCluster)
	}

	tokenCacheDuration := jsonConfig.Get(configKeyTokenCacheDuration).Uint()
	propagationHeader := jsonConfig.Get(configKeyOriginalAuthorizationPropagationHeader).String()

	return &pluginConfiguration{
		audience:                               audience,
		metadataServerCluster:                  cluster,
		tokenCacheDuration:                     tokenCacheDuration,
		originalAuthorizationPropagationHeader: propagationHeader,
	}, nil
}
