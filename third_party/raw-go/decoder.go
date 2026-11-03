package raw

import (
	"fmt"
	"image"
	"image/color"
	"io"
)

func init() {
	// Register the magic numbers that unambiguously identify a RAW file so that image.Decode / image.DecodeConfig can
	// auto-detect them.
	//
	// TIFF-based formats (NEF, ARW, DNG, ORF, ...) share the plain TIFF magic ("II*\0" / "MM\0*") with regular TIFF
	// images and would collide with the standard library's tiff decoder, so they are intentionally NOT registered here
	// — decode those by calling raw.Decode directly.
	image.RegisterFormat("raw-cr2", "II*\x00\x10\x00\x00\x00CR", decode, decodeConfigReader)
	image.RegisterFormat("raw-raf", "FUJIFILMCCD-RAW", decode, decodeConfigReader)
}

// Decode reads a camera RAW image from r and returns it as a fully demosaiced image.Image (8-bit RGBA). It works for
// any RAW format supported by LibRaw, regardless of file extension.
//
// By default it applies the camera's as-shot white balance and auto-brightness, producing a natural rendering close
// to the camera's own JPEG. Pass Option values (WithWhiteBalance, WithAutoBright, WithBrightness) to tune the result.
func Decode(r io.Reader, opts ...Option) (image.Image, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("raw: failed to read input: %w", err)
	}

	img, err := decodeRaw(data, buildOptions(opts))
	if err != nil {
		return nil, fmt.Errorf("raw: failed to decode image: %w", err)
	}

	return img, nil
}

// DecodeConfig reads the dimensions and color model of a RAW image from r without demosaicing the full image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return image.Config{}, fmt.Errorf("raw: failed to read input: %w", err)
	}

	cfg, err := decodeConfig(data)
	if err != nil {
		return image.Config{}, fmt.Errorf("raw: failed to read config: %w", err)
	}

	cfg.ColorModel = color.RGBAModel
	return cfg, nil
}

// decode adapts Decode to the signature expected by image.RegisterFormat.
func decode(r io.Reader) (image.Image, error) {
	return Decode(r)
}

// decodeConfigReader adapts DecodeConfig to the signature expected by image.RegisterFormat.
func decodeConfigReader(r io.Reader) (image.Config, error) {
	return DecodeConfig(r)
}
