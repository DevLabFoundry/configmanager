// Package plugin
// provides reactor style module
// we can explore the plugin provided host module
package plugin

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/DevLabFoundry/configmanager/v3/internal/config"
	"github.com/DevLabFoundry/configmanager/v3/plugins"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var (
	ErrMissingMethod         = errors.New("missing method on the wasiLib instance")
	ErrAllocMemForParam      = errors.New("failed to allocate memory for property")
	ErrAllocateOutPtrZeroLen = errors.New("allocate returned 0 for output pointer")
	ErrMemoryReadFailed      = errors.New("mem.Read(out) failed")
	ErrEmptyToken            = errors.New("token must not be empty")
)

// ====================
// Engine & ApiInstance
// ====================

type Engine struct {
	r              wazero.Runtime
	compiledModule wazero.CompiledModule
}

// NewEngine compiles the WASI module once for the lifetime of the program.
func NewEngine(ctx context.Context, ps io.ReadCloser) (*Engine, error) {
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("instantiate WASI: %w", err)
	}

	defer ps.Close()
	wasiLib, err := io.ReadAll(ps)
	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("read plugin: %w", err)
	}

	cm, err := r.CompileModule(ctx, wasiLib)
	if err != nil {
		_ = r.Close(ctx)
		return nil, fmt.Errorf("compile module: %w", err)
	}

	return &Engine{
		r:              r,
		compiledModule: cm,
	}, nil
}

// Close shuts down the runtime.
func (e *Engine) Close(ctx context.Context) error {
	return e.r.Close(ctx)
}

type ApiInstance struct {
	mod api.Module
	mem api.Memory
	// exported alloc helpers
	allocate   api.Function
	deallocate api.Function
	// exported business function
	tokenValue api.Function
	// scratch output buffers
	outPtr    uint32
	outCap    uint32
	outLenPtr uint32 // 4-byte cell for required length
}

// NewApiInstance instantiates a fresh module instance.
func (e *Engine) NewApiInstance(ctx context.Context) (*ApiInstance, error) {
	mod, err := e.r.InstantiateModule(ctx, e.compiledModule, wazero.NewModuleConfig().WithStartFunctions("_initialize"))
	if err != nil {
		return nil, fmt.Errorf("instantiate module: %w", err)
	}

	inst := &ApiInstance{
		mod:        mod,
		mem:        mod.Memory(),
		allocate:   mod.ExportedFunction("allocate"),
		deallocate: mod.ExportedFunction("deallocate"),
		tokenValue: mod.ExportedFunction("strategy_token_value"),
	}

	for name, exported := range map[string]api.Function{
		"allocate":             inst.allocate,
		"deallocate":           inst.deallocate,
		"strategy_token_value": inst.tokenValue,
	} {
		if exported == nil {
			return nil, fmt.Errorf("%w, method %q not found on exports", ErrMissingMethod, name)
		}
	}

	return inst, nil
}

// Close instance (optional).
func (i *ApiInstance) Close(ctx context.Context) {
	i.freeScratch(ctx)
	_ = i.mod.Close(ctx)
}

// put allocates module memory and writes bytes into it.
// returns (ptr, size). caller must deallocate(ptr, size).
func (i *ApiInstance) put(ctx context.Context, b []byte) (uint32, uint32, error) {
	if len(b) == 0 {
		return 0, 0, ErrEmptyToken
	}

	res, err := i.allocate.Call(ctx, uint64(len(b)))
	if err != nil {
		return 0, 0, fmt.Errorf("allocate: %w", err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, 0, fmt.Errorf("allocate returned 0: %w", ErrAllocMemForParam)
	}

	if ok := i.mem.Write(ptr, b); !ok {
		_, _ = i.deallocate.Call(ctx, uint64(ptr), uint64(len(b)))
		return 0, 0, fmt.Errorf("mem.Write failed: %w", ErrAllocMemForParam)
	}

	return ptr, uint32(len(b)), nil
}

// ensureOut ensures the scratch output buffer has at least `need` bytes.
// allocates outLenPtr (4 bytes) once.
func (i *ApiInstance) ensureOut(ctx context.Context, need uint32) error {
	// outLenPtr is a 4-byte cell for required length
	if i.outLenPtr == 0 {
		res, err := i.allocate.Call(ctx, 4)
		if err != nil {
			return fmt.Errorf("allocate outLenPtr: %w", err)
		}
		i.outLenPtr = uint32(res[0])
		if i.outLenPtr == 0 {
			return ErrAllocateOutPtrZeroLen
		}
	}

	if need <= i.outCap {
		return nil
	}

	// grow if needed - free old and alloc new
	if i.outPtr != 0 {
		_, _ = i.deallocate.Call(ctx, uint64(i.outPtr), uint64(i.outCap))
		i.outPtr, i.outCap = 0, 0
	}

	res, err := i.allocate.Call(ctx, uint64(need))
	if err != nil {
		return fmt.Errorf("allocate outPtr: %w", err)
	}
	i.outPtr, i.outCap = uint32(res[0]), need
	if i.outPtr == 0 {
		return ErrAllocateOutPtrZeroLen
	}
	return nil
}

// freeScratch frees the reusable output buffers (call once per instance).
func (i *ApiInstance) freeScratch(ctx context.Context) {
	if i.outPtr != 0 {
		_, _ = i.deallocate.Call(ctx, uint64(i.outPtr), uint64(i.outCap))
		i.outPtr, i.outCap = 0, 0
	}
	if i.outLenPtr != 0 {
		_, _ = i.deallocate.Call(ctx, uint64(i.outLenPtr), 4)
		i.outLenPtr = 0
	}
}

// TokenValue is the nice host-side API: string in, []byte out.
func (i *ApiInstance) TokenValue(ctx context.Context, token *config.ParsedTokenConfig) ([]byte, error) {
	if token.StoreToken() == "" {
		return nil, ErrEmptyToken
	}
	tokenBytes, err := token.JSONMessagExchangeBytes()
	tokenPtr, tokenLen, err := i.put(ctx, tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("put input: %w", err)
	}
	defer i.deallocate.Call(ctx, uint64(tokenPtr), uint64(tokenLen))

	// start with a smallish buffer; plugin will ask for more if needed
	if err := i.ensureOut(ctx, 64); err != nil {
		return nil, fmt.Errorf("ensureOut: %w", err)
	}

	call := func() (int32, uint32, error) {
		res, err := i.tokenValue.Call(
			ctx,
			uint64(tokenPtr), uint64(tokenLen), // sanitizedToken
			uint64(i.outPtr), uint64(i.outCap), // outPtr, outCap
			uint64(i.outLenPtr), // outLenPtr
		)
		if err != nil {
			return 0, 0, fmt.Errorf("call strategy_token_value: %w", err)
		}

		lenBytes, ok := i.mem.Read(i.outLenPtr, 4)
		if !ok {
			return int32(res[0]), 0, ErrMemoryReadFailed
		}

		need := binary.LittleEndian.Uint32(lenBytes)
		return int32(res[0]), need, nil
	}

	rc, need, err := call()
	if err != nil {
		return nil, err
	}

	if rc == plugins.ERR_BUF_TOO_SMALL {
		if err := i.ensureOut(ctx, need); err != nil {
			return nil, fmt.Errorf("ensureOut resize: %w", err)
		}
		rc, need, err = call()
		if err != nil {
			return nil, err
		}
	}

	if rc != plugins.OK {
		switch rc {
		case plugins.ERR_INVALID_UTF8:
			return nil, errors.New("token value: invalid UTF-8 in input")
		case plugins.ERR_EMPTY_INPUT:
			return nil, ErrEmptyToken
		case plugins.ERR_BUF_TOO_SMALL:
			return nil, fmt.Errorf("token value: buffer too small even after resize (need=%d)", need)
		default:
			return nil, fmt.Errorf("token value: unknown error code %d", rc)
		}
	}

	out, ok := i.mem.Read(i.outPtr, need)
	if !ok {
		return nil, ErrMemoryReadFailed
	}

	// Detach from wasm memory.
	result := make([]byte, need)
	copy(result, out)
	return result, nil
}
