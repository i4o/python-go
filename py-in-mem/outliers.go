package outliers

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"
)

/*
#cgo pkg-config: python3
#cgo LDFLAGS: -lpython3.8

#include "glue.h"

*/
import "C"

var (
	initOnce sync.Once
	initErr  error
)

func initialize() {
	initOnce.Do(func() {
		C.init_python()
		initErr = pyLastError()
	})
}

// Outliers does outlier detection
type Outliers struct {
	pyFunc *C.PyObject
}

// NewOutliers returns an new Outliers using moduleName.funcName Python function
func NewOutliers(moduleName, funcName string) (*Outliers, error) {
	initialize()
	if initErr != nil {
		return nil, initErr
	}
	pyFunc, err := loadPyFunc(moduleName, funcName)
	if err != nil {
		return nil, err
	}

	out := &Outliers{pyFunc}
	runtime.SetFinalizer(out, func(o *Outliers) {
		C.py_decref(out.pyFunc)
	})

	return out, nil
}

// Detect returns slice of outliers indices
func (o *Outliers) Detect(data []float64) ([]int, error) {
	carr := (*C.double)(&(data[0]))
	res := C.detect(o.pyFunc, carr, (C.long)(len(data)))
	runtime.KeepAlive(data)
	if res.err != 0 {
		return nil, pyLastError()
	}

	// Convert C int* to []int
	ptr := unsafe.Pointer(res.indices)
	// Ugly hack to convert C.long* to []int
	arr := (*[1 << 20]int)(ptr)
	indices := arr[:res.size]
	/* FIXME
	runtime.SetFinalizer(indices, func() {
		C.free(ptr)
	})
	*/
	return indices, nil
}

func loadPyFunc(moduleName, funcName string) (*C.PyObject, error) {
	cMod := C.CString(moduleName)
	cFunc := C.CString(funcName)
	defer func() {
		C.free(unsafe.Pointer(cMod))
		C.free(unsafe.Pointer(cFunc))
	}()

	pyFunc := C.load_func(cMod, cFunc)
	if pyFunc == nil {
		return nil, pyLastError()
	}

	return pyFunc, nil
}

func pyLastError() error {
	cp := C.py_last_error()
	if cp == nil {
		return nil
	}

	err := C.GoString(cp)
	// C.free(unsafe.Pointer(cp)) // FIXME
	return fmt.Errorf("%s", err)
}
