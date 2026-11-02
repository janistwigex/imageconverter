package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/draw"
	raw "github.com/vegidio/raw-go"
)

var rawExtensions = map[string]bool{
	".cr2": true, ".cr3": true, ".nef": true, ".arw": true, ".dng": true,
	".raf": true, ".orf": true, ".rw2": true, ".pef": true, ".srw": true,
	".raw": true, ".crw": true, ".nrw": true, ".sr2": true, ".srf": true,
	".mrw": true, ".erf": true, ".kdc": true, ".dcr": true, ".3fr": true,
	".fff": true, ".iiq": true, ".rwl": true, ".x3f": true,
}

type config struct {
	Input       string   `json:"input"`
	Output      string   `json:"output"`
	Size        int      `json:"size"`
	Quality     int      `json:"quality"`
	Workers     int      `json:"workers"`
	Extensions  []string `json:"extensions"`
	Force       bool     `json:"force"`
	DryRun      bool     `json:"dry_run"`
}

type task struct {
	src string
	dst string
}

func loadConfig(path string) (config, error) {
	var c config
	if path == "" {
		return c, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return c, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, err
	}
	return c, nil
}

func findConfig() string {
	candidates := []string{"config.json", filepath.Join(filepath.Dir(os.Args[0]), "config.json")}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func hasExt(name string, exts map[string]bool) bool {
	return exts[strings.ToLower(filepath.Ext(name))]
}

func collectTasks(input, output string, exts map[string]bool) ([]task, error) {
	var tasks []task
	outAbs, _ := filepath.Abs(output)
	err := filepath.WalkDir(input, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			abs, _ := filepath.Abs(path)
			if abs == outAbs {
				return filepath.SkipDir
			}
			if strings.HasPrefix(abs, outAbs+string(os.PathSeparator)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !hasExt(d.Name(), exts) {
			return nil
		}
		rel, err := filepath.Rel(input, path)
		if err != nil {
			return err
		}
		base := strings.TrimSuffix(rel, filepath.Ext(rel))
		dstRel := base + ".jpg"
		dst := filepath.Join(output, dstRel)
		tasks = append(tasks, task{src: path, dst: dst})
		return nil
	})
	return tasks, err
}

func resizeImage(img image.Image, maxDim int) image.Image {
	if maxDim <= 0 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxDim && h <= maxDim {
		return img
	}
	var nw, nh int
	if w >= h {
		nw = maxDim
		nh = int(float64(h) * float64(maxDim) / float64(w))
	} else {
		nh = maxDim
		nw = int(float64(w) * float64(maxDim) / float64(h))
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	return dst
}

// extractExifSegment pulls the first APP1/Exif segment (including marker) from JPEG bytes.
func extractExifSegment(data []byte) []byte {
	i := 2 // after SOI
	for i < len(data) {
		if data[i] != 0xFF {
			i++
			continue
		}
		j := i
		for j < len(data) && data[j] == 0xFF {
			j++
		}
		if j >= len(data) {
			break
		}
		marker := data[j]
		if marker == 0xD9 { // EOI
			break
		}
		if marker == 0xD8 { // SOI
			j++
			i = j
			continue
		}
		if marker >= 0xD0 && marker <= 0xD7 { // RST
			j++
			i = j
			continue
		}
		if j+3 > len(data) {
			break
		}
		length := int(data[j+1])<<8 | int(data[j+2])
		segStart := i
		segEnd := i + 2 + length
		if segEnd > len(data) {
			break
		}
		if marker == 0xE1 && segEnd-segStart >= 2+length {
			payload := data[j+3 : segEnd]
			if bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				return data[segStart:segEnd]
			}
		}
		i = segEnd
	}
	return nil
}

func injectExif(jpegBytes, exifSeg []byte) []byte {
	if len(exifSeg) == 0 {
		return jpegBytes
	}
	out := make([]byte, 0, len(jpegBytes)+len(exifSeg))
	out = append(out, jpegBytes[:2]...) // SOI
	out = append(out, exifSeg...)
	out = append(out, jpegBytes[2:]...)
	return out
}

func suppressStderr() func() {
	old := os.Stderr
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return func() {}
	}
	os.Stderr = devnull
	return func() {
		os.Stderr = old
		devnull.Close()
	}
}

var rawMu sync.Mutex

func convertOne(src, dst string, maxDim, quality int, force bool) (string, error) {
	if !force {
		if info, err := os.Stat(dst); err == nil && info.Size() > 0 {
			return "skipped", nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return "error", err
	}

	restore := suppressStderr()
	defer restore()

	f, err := os.Open(src)
	if err != nil {
		return "error", err
	}
	defer f.Close()

	rawMu.Lock()
	img, err := raw.Decode(f)
	var exifSeg []byte
	if err == nil {
		if tf, err2 := os.Open(src); err2 == nil {
			if tb, err3 := raw.ExtractThumbnail(tf); err3 == nil {
				exifSeg = extractExifSegment(tb)
			}
			tf.Close()
		}
	}
	rawMu.Unlock()
	if err != nil {
		return "error", err
	}

	img = resizeImage(img, maxDim)

	tmp := dst + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return "error", err
	}
	if err := jpeg.Encode(out, img, &jpeg.Options{Quality: quality}); err != nil {
		out.Close()
		os.Remove(tmp)
		return "error", err
	}
	out.Close()

	// inject exif if available
	if len(exifSeg) > 0 {
		b, err := os.ReadFile(tmp)
		if err == nil {
			b = injectExif(b, exifSeg)
			os.WriteFile(tmp, b, 0644)
		}
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return "error", err
	}
	return "ok", nil
}

func strSliceToExtSet(list []string) map[string]bool {
	if len(list) == 0 {
		return rawExtensions
	}
	m := map[string]bool{}
	for _, e := range list {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		m[e] = true
	}
	if len(m) == 0 {
		return rawExtensions
	}
	return m
}

func main() {
	configPath := flag.String("config", "", "Path to JSON config file")
	input := flag.String("input", "", "Input root path")
	output := flag.String("output", "", "Output root path")
	size := flag.Int("size", -1, "Max dimension in pixels (0 = no resize, -1 = use config/default)")
	quality := flag.Int("quality", 0, "JPEG quality 1-100 (0 = use default/config)")
	workers := flag.Int("workers", 0, "Parallel workers (0 = CPU count)")
	extsFlag := flag.String("extensions", "", "Comma-separated RAW extensions")
	force := flag.Bool("force", false, "Reprocess existing outputs")
	dryRun := flag.Bool("dry-run", false, "List files without converting")
	verbose := flag.Bool("verbose", false, "Per-file progress")
	logFile := flag.String("log-file", "", "Write log to file")
	flag.Parse()

	if *configPath == "" {
		*configPath = findConfig()
	}
	cfg, _ := loadConfig(*configPath)

	if *input == "" && cfg.Input != "" {
		*input = cfg.Input
	}
	if *output == "" && cfg.Output != "" {
		*output = cfg.Output
	}
	if *size < 0 {
		if cfg.Size != 0 {
			*size = cfg.Size
		} else {
			*size = 0
		}
	}
	if *quality == 0 {
		if cfg.Quality != 0 {
			*quality = cfg.Quality
		} else {
			*quality = 85
		}
	}
	if *workers == 0 {
		if cfg.Workers != 0 {
			*workers = cfg.Workers
		}
	}
	if *workers <= 0 {
		*workers = runtime.NumCPU()
	}
	var exts map[string]bool
	if *extsFlag != "" {
		parts := strings.Split(*extsFlag, ",")
		exts = strSliceToExtSet(parts)
	} else if len(cfg.Extensions) > 0 {
		exts = strSliceToExtSet(cfg.Extensions)
	} else {
		exts = rawExtensions
	}
	if *force == false && cfg.Force {
		*force = true
	}
	if *dryRun == false && cfg.DryRun {
		*dryRun = true
	}

	if *input == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --input and --output are required (or set in config).")
		os.Exit(2)
	}
	if *quality < 1 || *quality > 100 {
		fmt.Fprintln(os.Stderr, "ERROR: quality must be 1-100")
		os.Exit(2)
	}

	// logging
	logOut := os.Stdout
	if *logFile != "" {
		f, err := os.Create(*logFile)
		if err == nil {
			logOut = f
		}
	}
	logf := func(format string, args ...any) {
		fmt.Fprintf(logOut, format+"\n", args...)
	}
	vlog := func(format string, args ...any) {
		if *verbose {
			logf(format, args...)
		}
	}

	logf("Input : %s", *input)
	logf("Output: %s", *output)
	if *size > 0 {
		logf("Size  : max %dpx", *size)
	} else {
		logf("Size  : original")
	}
	logf("Quality: %d", *quality)
	logf("Workers: %d", *workers)

	if _, err := os.Stat(*input); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: input path not found: %s\n", *input)
		os.Exit(2)
	}

	tasks, err := collectTasks(*input, *output, exts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(2)
	}
	logf("Found %d RAW files to consider.", len(tasks))

	if *dryRun {
		for _, t := range tasks {
			status := "CONVERT"
			if !*force {
				if info, err := os.Stat(t.dst); err == nil && info.Size() > 0 {
					status = "SKIP"
				}
			}
			fmt.Printf("%s: %s -> %s\n", status, t.src, t.dst)
		}
		return
	}
	if len(tasks) == 0 {
		logf("Nothing to do.")
		return
	}

	maxDim := *size
	if maxDim <= 0 {
		maxDim = 0
	}

	var ok, skipped, errors int
	var mu sync.Mutex
	start := time.Now()
	total := len(tasks)

	jobs := make(chan task, total)
	for _, t := range tasks {
		jobs <- t
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				status, err := convertOne(t.src, t.dst, maxDim, *quality, *force)
				mu.Lock()
				switch status {
				case "ok":
					ok++
					vlog("OK   : %s", t.src)
				case "skipped":
					skipped++
					vlog("SKIP : %s", t.src)
				default:
					errors++
					logf("ERR  : %s -> %v", t.src, err)
				}
				done := ok + skipped + errors
				if done%50 == 0 || done == total {
					logf("Progress: %d/%d (ok=%d skipped=%d errors=%d)", done, total, ok, skipped, errors)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	logf("Done in %.1fs. ok=%d skipped=%d errors=%d total=%d", elapsed.Seconds(), ok, skipped, errors, total)
}