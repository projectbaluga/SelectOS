//go:build linux

package main

import "unsafe"

func unsafePtr(b []byte) unsafe.Pointer { return unsafe.Pointer(&b[0]) }
func unsafePtrB(b *byte) unsafe.Pointer { return unsafe.Pointer(b) }
