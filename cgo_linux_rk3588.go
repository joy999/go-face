//go:build linux && rk3588

package goface

/*
#cgo CFLAGS: -I${SRCDIR}/third_party/inspireface/include
#cgo LDFLAGS: -L${SRCDIR}/third_party/inspireface/lib/linux_aarch64_rk3588 -L/usr/local/lib -L/usr/lib -L/usr/lib/aarch64-linux-gnu -lInspireFace
#cgo LDFLAGS: -L${SRCDIR}/third_party/inspireface/lib/linux_aarch64_rk3588 -L/usr/local/lib -L/usr/lib -L/usr/lib/aarch64-linux-gnu -lrknnrt
#cgo LDFLAGS: -lm -ldl
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/third_party/inspireface/lib/linux_aarch64_rk3588
#cgo LDFLAGS: -Wl,-rpath,/usr/lib/aarch64-linux-gnu
*/
import "C"
