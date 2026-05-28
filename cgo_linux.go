//go:build linux && amd64 && !rk3588

package goface

/*
#cgo CFLAGS: -I${SRCDIR}/third_party/inspireface/include
#cgo LDFLAGS: -L${SRCDIR}/third_party/inspireface/lib/linux_x86 -lInspireFace -lm -ldl
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/third_party/inspireface/lib/linux_x86
*/
import "C"
