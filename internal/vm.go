package internal

import (
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
)

var _ types.VMContext = (*VmContext)(nil)

type VmContext struct {
	types.DefaultVMContext
}

func (*VmContext) NewPluginContext(contextID uint32) types.PluginContext {
	return &pluginContext{}
}
