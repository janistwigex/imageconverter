//go:build windows && amd64

package raw

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L./libs/windows_amd64 -lraw_r -ljasper -llcms2 -ljpeg -lzs -lstdc++ -lws2_32
*/
import "C"
