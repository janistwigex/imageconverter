package tests

import (
	"bytes"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	raw "github.com/vegidio/raw-go"
)

// rawExtensions are the file extensions recognized as RAW fixtures.
var rawExtensions = map[string]bool{
	".3fr": true, ".arw": true, ".cr2": true, ".cr3": true, ".crw": true,
	".dcr": true, ".dng": true, ".erf": true, ".fff": true, ".iiq": true,
	".kdc": true, ".mef": true, ".mos": true, ".mrw": true, ".nef": true,
	".nrw": true, ".orf": true, ".pef": true, ".raf": true, ".raw": true,
	".rw2": true, ".rwl": true, ".sr2": true, ".srf": true, ".srw": true,
	".x3f": true,
}

// fixtures returns the RAW files available in tests/testdata, skipping the test
// when none are present.
func fixtures(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("testdata")
	require.NoError(t, err)

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if rawExtensions[ext] || rawExtensions[lower(ext)] {
			files = append(files, filepath.Join("testdata", e.Name()))
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		t.Skip("no RAW fixtures in tests/testdata (see tests/testdata/README.md)")
	}
	return files
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func open(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestDecode(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			img, err := raw.Decode(open(t, path))
			require.NoError(t, err)
			require.NotNil(t, img)

			b := img.Bounds()
			assert.Positive(t, b.Dx())
			assert.Positive(t, b.Dy())
		})
	}
}

func TestDecodeConfig(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			cfg, err := raw.DecodeConfig(open(t, path))
			require.NoError(t, err)
			assert.Positive(t, cfg.Width)
			assert.Positive(t, cfg.Height)
			assert.NotNil(t, cfg.ColorModel)

			// DecodeConfig dimensions must match the full decode.
			img, err := raw.Decode(open(t, path))
			require.NoError(t, err)
			assert.Equal(t, cfg.Width, img.Bounds().Dx())
			assert.Equal(t, cfg.Height, img.Bounds().Dy())
		})
	}
}

func TestExtractThumbnail(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			thumb, err := raw.ExtractThumbnail(open(t, path))
			require.NoError(t, err)
			require.NotEmpty(t, thumb)

			// The result must always be a decodable JPEG.
			img, err := jpeg.Decode(bytes.NewReader(thumb))
			require.NoError(t, err)
			assert.Positive(t, img.Bounds().Dx())
			assert.Positive(t, img.Bounds().Dy())
		})
	}
}

func TestReadMetadata(t *testing.T) {
	for _, path := range fixtures(t) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			meta, err := raw.ReadMetadata(open(t, path))
			require.NoError(t, err)
			assert.NotEmpty(t, meta.Make)
			assert.NotEmpty(t, meta.Model)
			assert.Positive(t, meta.Width)
			assert.Positive(t, meta.Height)
		})
	}
}
