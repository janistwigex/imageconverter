//go:build linux && arm64

package raw

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L./libs/linux_arm64 -lraw_r -ljasper -llcms2 -ljpeg -lz -lstdc++ -lm
*/
import "C"
