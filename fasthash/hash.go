// Package fasthash exposes the fast hash functions in runtime package.
//
// The idea mainly comes from https://github.com/golang/go/issues/21195.
package fasthash

import (
	"reflect"
	"unsafe"
)

// Hash returns a hash code for a comparable argument using a hash function
// that is local to the current invocation of the program.
//
// Hash simply exposes the hash functions in runtime package. It will panic
// if x is nil or unsupported type by the runtime package.
//
// Each call to Hash with the same value will return the same result within the
// lifetime of the program, but each run of the program may return different
// results from previous runs.
func Hash(x interface{}) uintptr {
	switch v := x.(type) {
	case string:
		return String(v)
	case []byte:
		return Bytes(v)
	case int8:
		return Int8(v)
	case uint8:
		return Uint8(v)
	case int16:
		return Int16(v)
	case uint16:
		return Uint16(v)
	case int32:
		return Uint32(uint32(v))
	case uint32:
		return Uint32(v)
	case int64:
		return Uint64(uint64(v))
	case uint64:
		return Uint64(v)
	case int:
		return uintptrHash(uintptr(v))
	case uint:
		return uintptrHash(uintptr(v))
	case uintptr:
		return uintptrHash(v)
	case float32:
		return Float32(v)
	case float64:
		return Float64(v)
	case complex64:
		return Complex64(v)
	case complex128:
		return Complex128(v)
	default:
		return Interface(v)
	}
}

// String exposes the stringHash function from runtime package.
func String(x string) uintptr {
	return stringHash(x, seed)
}

// Bytes exposes the bytesHash function from runtime package.
func Bytes(x []byte) uintptr {
	return bytesHash(x, seed)
}

// Int8 exposes the memhash8 function from runtime package.
func Int8(x int8) uintptr {
	return memhash8(noescape(unsafe.Pointer(&x)), seed)
}

// Uint8 exposes the memhash8 function from runtime package.
func Uint8(x uint8) uintptr {
	return memhash8(noescape(unsafe.Pointer(&x)), seed)
}

// Int16 exposes the memhash16 function from runtime package.
func Int16(x int16) uintptr {
	return memhash16(noescape(unsafe.Pointer(&x)), seed)
}

// Uint16 exposes the memhash16 function from runtime package.
func Uint16(x uint16) uintptr {
	return memhash16(noescape(unsafe.Pointer(&x)), seed)
}

// Int32 exposes the int32Hash function from runtime package.
func Int32(x int32) uintptr {
	return int32Hash(uint32(x), seed)
}

// Uint32 exposes the int32Hash function from runtime package.
func Uint32(x uint32) uintptr {
	return int32Hash(x, seed)
}

// Int64 exposes the int64Hash function from runtime package.
func Int64(x int64) uintptr {
	return int64Hash(uint64(x), seed)
}

// Uint64 exposes the int64Hash function from runtime package.
func Uint64(x uint64) uintptr {
	return int64Hash(x, seed)
}

// Int calculates hash of x using either int32Hash or int64Hash
// according to the pointer size of the platform.
func Int(x int) uintptr {
	return uintptrHash(uintptr(x))
}

// Uint calculates hash of x using either int32Hash or int64Hash
// according the pointer size of the platform.
func Uint(x uint) uintptr {
	return uintptrHash(uintptr(x))
}

// Uintptr calculates hash of x using either int32Hash or int64Hash
// according to the pointer size of the platform.
func Uintptr(x uintptr) uintptr {
	return uintptrHash(x)
}

// Float32 exposes the f32hash function from runtime package.
func Float32(x float32) uintptr {
	return f32hash(noescape(unsafe.Pointer(&x)), seed)
}

// Float64 exposes the f64hash function from runtime package.
func Float64(x float64) uintptr {
	return f64hash(noescape(unsafe.Pointer(&x)), seed)
}

// Complex64 exposes the c64hash function from runtime package.
func Complex64(x complex64) uintptr {
	return c64hash(noescape(unsafe.Pointer(&x)), seed)
}

// Complex128 exposes the c128hash function from runtime package.
func Complex128(x complex128) uintptr {
	return c128hash(noescape(unsafe.Pointer(&x)), seed)
}

// Interface exposes the efaceHash function from runtime package.
func Interface(x interface{}) uintptr {
	return efaceHash(x, seed)
}

//go:linkname memhash8 runtime.memhash8
func memhash8(p unsafe.Pointer, h uintptr) uintptr

//go:linkname memhash16 runtime.memhash16
func memhash16(p unsafe.Pointer, h uintptr) uintptr

//go:linkname stringHash runtime.stringHash
func stringHash(s string, seed uintptr) uintptr

//go:linkname bytesHash runtime.bytesHash
func bytesHash(b []byte, seed uintptr) uintptr

//go:linkname int32Hash runtime.int32Hash
func int32Hash(i uint32, seed uintptr) uintptr

//go:linkname int64Hash runtime.int64Hash
func int64Hash(i uint64, seed uintptr) uintptr

//go:linkname f32hash runtime.f32hash
func f32hash(p unsafe.Pointer, h uintptr) uintptr

//go:linkname f64hash runtime.f64hash
func f64hash(p unsafe.Pointer, h uintptr) uintptr

//go:linkname c64hash runtime.c64hash
func c64hash(p unsafe.Pointer, h uintptr) uintptr

//go:linkname c128hash runtime.c128hash
func c128hash(p unsafe.Pointer, h uintptr) uintptr

//go:linkname efaceHash runtime.efaceHash
func efaceHash(i interface{}, seed uintptr) uintptr

//go:linkname getRandomData runtime.getRandomData
func getRandomData(r []byte)

// noescape is copied from the runtime package.
//
// noescape hides a pointer from escape analysis.  noescape is
// the identity function but escape analysis doesn't think the
// output depends on the input.  noescape is inlined and currently
// compiles down to zero instructions.
// USE CAREFULLY!
//go:nosplit
func noescape(p unsafe.Pointer) unsafe.Pointer {
	x := uintptr(p)
	return unsafe.Pointer(x ^ 0)
}

// intSize is the size in bits of an int or uint value.
const intSize = 32 << (^uint(0) >> 63)

var seed uintptr

var uintptrHash func(uintptr) uintptr

func init() {
	var tmp []byte
	ptr := (*reflect.SliceHeader)(unsafe.Pointer(&tmp))
	ptr.Data = uintptr(unsafe.Pointer(&seed))
	ptr.Cap = int(unsafe.Sizeof(seed))
	ptr.Len = ptr.Cap
	getRandomData(tmp)

	if intSize == 32 {
		uintptrHash = func(x uintptr) uintptr { return int32Hash(uint32(x), seed) }
	} else {
		uintptrHash = func(x uintptr) uintptr { return int64Hash(uint64(x), seed) }
	}
}
