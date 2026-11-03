//go:build darwin && arm64

package raw

/*
#cgo CFLAGS: -I./include
#cgo LDFLAGS: -L./libs/darwin_arm64 -lraw_r -ljasper -llcms2 -ljpeg -lz -lc++
*/
import "C"
