package darwin

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreBluetooth

#import "bt.h"
*/
import "C"

import "unsafe"

func mallocArr(numElems int, elemSize uintptr) unsafe.Pointer {
	return C.malloc(C.size_t(numElems) * C.size_t(elemSize))
}

// cArrGetAddr retrieves the *address* of an element from a C array.
func cArrGetAddr(arr unsafe.Pointer, elemSize uintptr, idx int) unsafe.Pointer {
	base := uintptr(arr)
	off := uintptr(idx) * elemSize
	cur := base + off
	return unsafe.Pointer(cur)
}
func byteArrToByteSlice(byteArr C.struct_byte_arr) []byte {
	return C.GoBytes(unsafe.Pointer(byteArr.data), byteArr.length)
}

func byteSliceToByteArr(b []byte) C.struct_byte_arr {
	if len(b) == 0 {
		return C.struct_byte_arr{
			data:   nil,
			length: 0,
		}
	} else {
		return C.struct_byte_arr{
			data:   (*C.uint8_t)(C.CBytes(b)),
			length: C.int(len(b)),
		}
	}
}

func stringSliceToArr(ss []string) **C.char {
	if len(ss) == 0 {
		return nil
	}

	ptr := mallocArr(len(ss), unsafe.Sizeof((*C.char)(nil)))

	carr := (*[1<<30 - 1]*C.char)(ptr)
	for i, s := range ss {
		carr[i] = C.CString(s)
	}

	return (**C.char)(ptr)
}

func byteSliceSliceToArr(bs [][]byte) *C.struct_byte_arr {
	dummyElem := C.struct_byte_arr{}
	elemSize := unsafe.Sizeof(dummyElem)
	ptr := C.malloc(C.size_t(len(bs)) * C.size_t(elemSize))

	carr := (*[1<<30 - 1]C.struct_byte_arr)(ptr)
	for i, b := range bs {
		carr[i] = byteSliceToByteArr(b)
	}

	return (*C.struct_byte_arr)(ptr)
}

// cArrGetStr retrieves an element from a C array of strings (i.e., char**) and
// converts it to a Go string.
func cArrGetStr(arr **C.char, idx int) string {
	dummy := (*C.char)(nil)
	elemSize := unsafe.Sizeof(dummy)

	ptr := cArrGetAddr(unsafe.Pointer(arr), elemSize, idx)
	cstr := *(**C.char)(ptr)
	return C.GoString(cstr)
}
