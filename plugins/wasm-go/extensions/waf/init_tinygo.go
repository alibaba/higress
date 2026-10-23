// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

//go:build tinygo

package main

import (
	"unsafe"
)

// Some host functions that are not implemented by Envoy end up getting imported anyways
// by code that gets compiled but not executed at runtime. Because we know they are not
// executed, we can stub them out to allow functioning on Envoy. Note, these match the
// names and signatures of wasi-libc, used by TinyGo, not WASI ABI. Review these exports when either
// the minimum supported version of Envoy changes or the maximum version of TinyGo.

// fdopendir is re-exported to avoid TinyGo 0.28's import of wasi_snapshot_preview1.fd_readdir.
//
//export fdopendir
func fdopendir(fd int32) unsafe.Pointer {
	return nil
}

// readdir is re-exported to avoid TinyGo 0.28's import of wasi_snapshot_preview1.fd_readdir.
//
//export readdir
func readdir(unsafe.Pointer) unsafe.Pointer {
	return nil
}
