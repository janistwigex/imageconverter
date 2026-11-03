//go:build windows && arm64

package raw

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L./libs/windows_arm64 -lraw_r -ljasper -llcms2 -ljpeg -lzs -lstdc++ -lws2_32
*/
import "C"
