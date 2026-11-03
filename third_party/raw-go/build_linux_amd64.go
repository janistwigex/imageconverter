//go:build linux && amd64

package raw

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L./libs/linux_amd64 -lraw_r -ljasper -llcms2 -ljpeg -lz -lstdc++ -lm
*/
import "C"
