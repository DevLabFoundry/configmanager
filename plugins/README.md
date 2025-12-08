# Configmanager Plugin System

The plugin architecture for configmanager is built using the [go-plugin](https://github.com/hashicorp/go-plugin?tab=readme-ov-file#go-plugin-system-over-rpc) from hashicorp. 


The existing implementations are converted into plugins using the gRPC model and are built using
gRPC [go-plugin](https://github.com/hashicorp/go-plugin?tab=readme-ov-file#go-plugin-system-over-rpc) and generated/updated with the [buf cli](https://buf.build/docs/cli/).


## Plugin Architecture

<!-- TODO: add diagram -->
```mermaid
```

The plugins will need to be downloaded into any one of these locations on disk, they will be checked in this order

- currentDirectory (directory from which the configmanager executable is run)
- users home directory

The plugin is expected to be found under this path in the above locations 
> `.configmanager/plugins/$PLUGIN_PREFIX_LOWERCASE/$PLUGIN_PREFIX_LOWERCASE-$GOOS-$GOARCH` 

e.g. in case of the AWS Parameter Store plugin `.configmanager/plugins/awsparamstr/awsparamstr-linux-amd64`



## Alternate architecture explored

As part of the decision on which pluging architecture to use we also explored an alternate architecture using WASIP1.



```go
import (
	"context"
	"encoding/binary"
	"encoding/json"
	"sync"
	"unicode/utf8"
	"unsafe"

	"github.com/DevLabFoundry/configmanager/v3/plugins"
)

// ====================
// Bump allocator
// ====================

const heapSize = 64 * 1024 // 64 KiB arena; tune as needed

type bumpAllocator struct {
	mu   sync.Mutex
	heap []byte
	used uint32
}

var alloc = bumpAllocator{
	heap: make([]byte, heapSize),
}

// round allocation up to 8 bytes for basic alignment.
func roundUp(n uint32) uint32 {
	const align = 8
	return (n + align - 1) &^ (align - 1)
}

//go:wasmexport allocate
func Allocate(size uint32) uint32 {
	if size == 0 {
		return 0
	}
	size = roundUp(size)

	alloc.mu.Lock()
	defer alloc.mu.Unlock()

	if alloc.used+size > uint32(len(alloc.heap)) {
		// Out of memory in our arena.
		return 0
	}

	offset := alloc.used
	alloc.used += size

	// Return pointer into linear memory for &heap[offset].
	return uint32(uintptr(unsafe.Pointer(&alloc.heap[offset])))
}

//go:wasmexport deallocate
func Deallocate(ptr, size uint32) {
	// For a simple bump allocator, deallocate is a no-op.
	// Memory is reclaimed when the module instance is destroyed.
	_ = ptr
	_ = size
}

type Hdr struct {
	Data uintptr
	Len  int
	Cap  int
}

// ====================
// Helpers
// ====================

// bytesFromPtrLen reinterprets a (ptr,len) pair in wasm linear memory
// as a Go []byte without copying.
func bytesFromPtrLen(ptr, length uint32) []byte {
	if length == 0 {
		return nil
	}

	hdr := Hdr{
		Data: uintptr(ptr),
		Len:  int(length),
		Cap:  int(length),
	}

	return *(*[]byte)(unsafe.Pointer(&hdr))
}

// ====================
// strategy_token_value
// ====================
//
// ABI:
//
//	strategy_token_value(
//	    inPtr, inLen, outPtr, outCap, outLenPtr uint32,
//	) int32
//
// Host contract:
//   - Input bytes are at (inPtr, inLen)
//   - Output buffer is [outPtr : outPtr+outCap)
//   - outLenPtr points to 4 bytes where we write the required length
//
// Behaviour:
//   - If input length == 0 => ERR_EMPTY_INPUT
//   - If invalid UTF-8 => ERR_INVALID_UTF8
//   - Always write required length to *outLenPtr (little-endian)
//   - If required > outCap => ERR_BUF_TOO_SMALL
//   - Else copy into caller buffer and return OK
//
//go:wasmexport strategy_token_value
func StrategyTokenValue(tokenPtr, tokenLen, outPtr, outCap, outLenPtr uint32) int32 {
	defer func() {
		// Make sure panics don't leak as traps.
		if r := recover(); r != nil {
			if outLenPtr != 0 {
				if lenCell := bytesFromPtrLen(outLenPtr, 4); len(lenCell) == 4 {
					binary.LittleEndian.PutUint32(lenCell, 0)
				}
			}
		}
	}()

	if tokenLen == 0 {
		// Mark required length as 0 and signal error.
		if outLenPtr != 0 {
			if lenCell := bytesFromPtrLen(outLenPtr, 4); len(lenCell) == 4 {
				binary.LittleEndian.PutUint32(lenCell, 0)
			}
		}
		return plugins.ERR_EMPTY_INPUT
	}

	tokenBytes := bytesFromPtrLen(tokenPtr, tokenLen)
	if !utf8.Valid(tokenBytes) {
		if outLenPtr != 0 {
			if lenCell := bytesFromPtrLen(outLenPtr, 4); len(lenCell) == 4 {
				binary.LittleEndian.PutUint32(lenCell, uint32(len(tokenBytes)))
			}
		}
		return plugins.ERR_INVALID_UTF8
	}

	// --- Business logic (replace with your real token strategy) ---
	// unmarshal string into an object
	token := &plugins.MessagExchange{}
	if err := json.Unmarshal(tokenBytes, token); err != nil {
		return plugins.ERR_FAILED_UNMARSHAL_MESSAGE
	}

	// logger := log.New(os.Stdout)
	// logger.SetLevel(log.DebugLvl)

	store, err := NewParamStore(context.Background())
	if err != nil {
		return plugins.ERR_INIT_STORE
	}

	outStr, err := store.Value(token)
	if err != nil {
		return plugins.ERR_FAILED_VALUE_RETRIEVAL
	}

	outBytes := []byte(outStr)
	// --------------------------------------------------------------
	// BEGIN RETURN Allocation
	// --------------------------------------------------------------
	required := uint32(len(outBytes))

	// Always write required length.
	if outLenPtr != 0 {
		lenCell := bytesFromPtrLen(outLenPtr, 4)
		if len(lenCell) != 4 {
			return plugins.ERR_INTERNAL
		}
		binary.LittleEndian.PutUint32(lenCell, required)
	}

	if required > outCap {
		return plugins.ERR_BUF_TOO_SMALL
	}

	if required == 0 {
		return plugins.OK
	}

	outSlice := bytesFromPtrLen(outPtr, outCap)
	if uint32(len(outSlice)) < required {
		return plugins.ERR_INTERNAL
	}

	copy(outSlice, outBytes)
	return plugins.OK
}

// main is required for wasip1
// scaffolds the `_initialize` method
func main() {}
```

### Build notes

build using the `-buildmode=c-shared` which will convert the module to a reactor module

`GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o awsparams.wasm`
