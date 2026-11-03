package raw

import (
	"fmt"
	"io"
	"time"
)

// Metadata holds the camera information parsed from a RAW file.
type Metadata struct {
	Make      string    // camera manufacturer, e.g. "Nikon"
	Model     string    // camera model, e.g. "D1"
	Width     int       // image width in pixels
	Height    int       // image height in pixels
	ISO       float64   // ISO speed
	Shutter   float64   // shutter speed in seconds
	Aperture  float64   // aperture (f-number)
	FocalLen  float64   // focal length in mm
	Timestamp time.Time // capture time (zero if unavailable)
}

// ReadMetadata reads the camera metadata of a RAW file from r without demosaicing the image.
func ReadMetadata(r io.Reader) (Metadata, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Metadata{}, fmt.Errorf("raw: failed to read input: %w", err)
	}

	return readMetadata(data)
}
