package raw

import (
	"fmt"
	"io"
)

// ExtractThumbnail returns the embedded preview of a RAW file encoded as JPEG.
//
// Most cameras store a JPEG preview, which is returned as-is. A few (typically older) cameras store an uncompressed RGB
// bitmap preview instead; in that case the bitmap is encoded to JPEG, so the returned bytes are always a valid JPEG.
func ExtractThumbnail(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("raw: failed to read input: %w", err)
	}

	thumb, err := extractThumbnail(data)
	if err != nil {
		return nil, fmt.Errorf("raw: failed to extract thumbnail: %w", err)
	}

	return thumb, nil
}
