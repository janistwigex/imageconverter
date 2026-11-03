// Package raw is a Go library to decode camera RAW images (CR2, CR3, NEF, ARW, DNG, RAF, ORF, RW2, and many more)
// without any system dependencies.
//
// It statically links LibRaw via CGO, so neither LibRaw nor any of its dependencies need to be installed on the target
// machine. Building requires CGO_ENABLED=1 and a C toolchain.
package raw

/*
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "libraw.h"

// result codes returned by the wrappers below
enum { RAW_OK = 0, RAW_EOPEN, RAW_EUNPACK, RAW_EPROCESS, RAW_EMEM };

// raw_image holds an 8-bit, interleaved-RGB bitmap (decoded image or bitmap thumbnail). The caller must free `data`
// with free() once copied out.
typedef struct {
    unsigned char *data;
    int            width;
    int            height;
    int            colors;
    int            size;
} raw_image;

// raw_meta mirrors the subset of LibRaw metadata exposed by the Go API.
typedef struct {
    char  make[64];
    char  model[64];
    int   width;
    int   height;
    float iso;
    float shutter;
    float aperture;
    float focal;
    long  timestamp;
} raw_meta;

// raw_opts carries the tunable demosaic parameters from the Go API into LibRaw.
typedef struct {
    int   use_camera_wb;   // 1 = apply the camera's as-shot white balance
    int   use_auto_wb;     // 1 = compute an automatic white balance
    int   no_auto_bright;  // 1 = disable the auto-brightness histogram stretch
    float bright;          // brightness multiplier (1.0 = neutral)
} raw_opts;

// copy_processed copies a libraw_processed_image_t payload into a freshly malloc'd buffer that Go can free
// independently of LibRaw.
static unsigned char *copy_processed(libraw_processed_image_t *img, int *outSize) {
    *outSize = (int)img->data_size;
    unsigned char *buf = (unsigned char *)malloc(img->data_size);
    if (buf != NULL) {
        memcpy(buf, img->data, img->data_size);
    }
    return buf;
}

// decode_raw opens the RAW buffer and demosaics it to an 8-bit sRGB bitmap.
static int decode_raw(const void *data, size_t len, raw_image *out, raw_opts opts) {
    libraw_data_t *lr = libraw_init(0);
    if (lr == NULL) return RAW_EOPEN;

    int rc = RAW_OK;
    int errc = 0;
    libraw_processed_image_t *img = NULL;

    if (libraw_open_buffer(lr, data, len) != LIBRAW_SUCCESS) { rc = RAW_EOPEN; goto done; }
    if (libraw_unpack(lr) != LIBRAW_SUCCESS)                 { rc = RAW_EUNPACK; goto done; }

    lr->params.output_bps = 8;                       // 8 bits per sample
    lr->params.output_color = 1;                     // sRGB
    lr->params.use_camera_wb = opts.use_camera_wb;   // white balance source (see raw_opts)
    lr->params.use_auto_wb = opts.use_auto_wb;
    lr->params.no_auto_bright = opts.no_auto_bright; // brightness handling
    lr->params.bright = opts.bright;                 // brightness multiplier

    if (libraw_dcraw_process(lr) != LIBRAW_SUCCESS) { rc = RAW_EPROCESS; goto done; }

    img = libraw_dcraw_make_mem_image(lr, &errc);
    if (img == NULL || img->type != LIBRAW_IMAGE_BITMAP) {
        rc = RAW_EMEM;
        goto done;
    }

    out->data = copy_processed(img, &out->size);
    out->width = img->width;
    out->height = img->height;
    out->colors = img->colors;
    if (out->data == NULL) rc = RAW_EMEM;

done:
    if (img != NULL) libraw_dcraw_clear_mem(img);
    libraw_close(lr);
    return rc;
}

// extract_thumb returns the embedded preview. *isJPEG is 1 when the bytes are a complete JPEG file, or 0 when they are
// an 8-bit RGB bitmap.
static int extract_thumb(const void *data, size_t len, raw_image *out, int *isJPEG) {
    libraw_data_t *lr = libraw_init(0);
    if (lr == NULL) return RAW_EOPEN;

    int rc = RAW_OK;
    int errc = 0;
    libraw_processed_image_t *th = NULL;

    if (libraw_open_buffer(lr, data, len) != LIBRAW_SUCCESS) { rc = RAW_EOPEN; goto done; }
    if (libraw_unpack_thumb(lr) != LIBRAW_SUCCESS)           { rc = RAW_EUNPACK; goto done; }

    th = libraw_dcraw_make_mem_thumb(lr, &errc);
    if (th == NULL) { rc = RAW_EMEM; goto done; }

    *isJPEG = (th->type == LIBRAW_IMAGE_JPEG) ? 1 : 0;
    out->data = copy_processed(th, &out->size);
    out->width = th->width;
    out->height = th->height;
    out->colors = th->colors;
    if (out->data == NULL) rc = RAW_EMEM;

done:
    if (th != NULL) libraw_dcraw_clear_mem(th);
    libraw_close(lr);
    return rc;
}

// read_meta reads dimensions and camera metadata without demosaicing.
static int read_meta(const void *data, size_t len, raw_meta *out) {
    libraw_data_t *lr = libraw_init(0);
    if (lr == NULL) return RAW_EOPEN;

    int rc = RAW_OK;
    if (libraw_open_buffer(lr, data, len) != LIBRAW_SUCCESS) { rc = RAW_EOPEN; goto done; }

    snprintf(out->make, sizeof(out->make), "%s", lr->idata.make);
    snprintf(out->model, sizeof(out->model), "%s", lr->idata.model);
    out->width = lr->sizes.iwidth;
    out->height = lr->sizes.iheight;
    out->iso = lr->other.iso_speed;
    out->shutter = lr->other.shutter;
    out->aperture = lr->other.aperture;
    out->focal = lr->other.focal_len;
    out->timestamp = (long)lr->other.timestamp;

done:
    libraw_close(lr);
    return rc;
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"strings"
	"time"
	"unsafe"
)

// librawError turns a wrapper result code into a descriptive Go error.
func librawError(code C.int) error {
	switch code {
	case C.RAW_EOPEN:
		return fmt.Errorf("libraw: could not open or recognize the RAW data")
	case C.RAW_EUNPACK:
		return fmt.Errorf("libraw: could not unpack the RAW data")
	case C.RAW_EPROCESS:
		return fmt.Errorf("libraw: could not process the RAW data")
	case C.RAW_EMEM:
		return fmt.Errorf("libraw: could not allocate the decoded image")
	default:
		return fmt.Errorf("libraw: unknown error (code %d)", int(code))
	}
}

// decodeRaw demosaics the RAW bytes into an 8-bit RGBA image using the supplied options.
func decodeRaw(data []byte, o options) (*image.RGBA, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("raw: empty input")
	}

	copts := C.raw_opts{
		use_camera_wb:  boolToCInt(o.whiteBalance == WhiteBalanceCamera),
		use_auto_wb:    boolToCInt(o.whiteBalance == WhiteBalanceAuto),
		no_auto_bright: boolToCInt(!o.autoBright),
		bright:         C.float(o.brightness),
	}

	var out C.raw_image
	rc := C.decode_raw(unsafe.Pointer(&data[0]), C.size_t(len(data)), &out, copts)
	if rc != C.RAW_OK {
		return nil, librawError(rc)
	}
	defer C.free(unsafe.Pointer(out.data))

	return rgbToRGBA(out)
}

// boolToCInt maps a Go bool to the 0/1 C int that LibRaw's parameter flags expect.
func boolToCInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// rgbToRGBA converts an interleaved 8-bit RGB C buffer into an *image.RGBA.
func rgbToRGBA(out C.raw_image) (*image.RGBA, error) {
	w, h, colors := int(out.width), int(out.height), int(out.colors)
	if w <= 0 || h <= 0 || colors != 3 {
		return nil, fmt.Errorf("raw: unexpected bitmap geometry (%dx%d, %d colors)", w, h, colors)
	}

	src := unsafe.Slice((*byte)(unsafe.Pointer(out.data)), int(out.size))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i, p := 0, 0; i+2 < len(src); i, p = i+3, p+4 {
		img.Pix[p] = src[i]     // R
		img.Pix[p+1] = src[i+1] // G
		img.Pix[p+2] = src[i+2] // B
		img.Pix[p+3] = 0xFF     // A
	}
	return img, nil
}

// decodeConfig reads the image dimensions without demosaicing.
func decodeConfig(data []byte) (image.Config, error) {
	m, err := readMetadata(data)
	if err != nil {
		return image.Config{}, err
	}
	return image.Config{
		ColorModel: nil, // set by the caller (color.RGBAModel)
		Width:      m.Width,
		Height:     m.Height,
	}, nil
}

// extractThumbnail returns the embedded preview encoded as JPEG. JPEG previews are returned as-is; bitmap previews are
// re-encoded, so the result is always a valid JPEG.
func extractThumbnail(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("raw: empty input")
	}

	var out C.raw_image
	var isJPEG C.int
	rc := C.extract_thumb(unsafe.Pointer(&data[0]), C.size_t(len(data)), &out, &isJPEG)
	if rc != C.RAW_OK {
		return nil, librawError(rc)
	}
	defer C.free(unsafe.Pointer(out.data))

	src := unsafe.Slice((*byte)(unsafe.Pointer(out.data)), int(out.size))

	if isJPEG != 0 {
		// Already a JPEG file: copy the bytes out of C memory and return them.
		thumb := make([]byte, len(src))
		copy(thumb, src)
		return thumb, nil
	}

	// Bitmap preview: re-encode the RGB pixels as JPEG.
	img, err := rgbToRGBA(out)
	if err != nil {
		return nil, fmt.Errorf("raw: failed to read bitmap thumbnail: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("raw: failed to encode thumbnail: %w", err)
	}
	return buf.Bytes(), nil
}

// readMetadata reads the camera metadata of the RAW bytes without demosaicing.
func readMetadata(data []byte) (Metadata, error) {
	if len(data) == 0 {
		return Metadata{}, fmt.Errorf("raw: empty input")
	}

	var out C.raw_meta
	rc := C.read_meta(unsafe.Pointer(&data[0]), C.size_t(len(data)), &out)
	if rc != C.RAW_OK {
		return Metadata{}, fmt.Errorf("raw: failed to read metadata: %w", librawError(rc))
	}

	m := Metadata{
		Make:     strings.TrimSpace(C.GoString(&out.make[0])),
		Model:    strings.TrimSpace(C.GoString(&out.model[0])),
		Width:    int(out.width),
		Height:   int(out.height),
		ISO:      float64(out.iso),
		Shutter:  float64(out.shutter),
		Aperture: float64(out.aperture),
		FocalLen: float64(out.focal),
	}
	if out.timestamp > 0 {
		m.Timestamp = time.Unix(int64(out.timestamp), 0).UTC()
	}

	return m, nil
}
