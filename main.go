package main

import (
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"

	"github.com/kauche/proxy-wasm-google-metadata-identity-token/internal"
)

func main() {
	proxywasm.SetVMContext(&internal.VmContext{})
}
