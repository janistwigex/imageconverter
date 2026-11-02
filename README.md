# imageconverter

Convert camera RAW images to JPEG while preserving the input folder structure.

Written in Go. The binary bundles LibRaw via [raw-go](https://github.com/vegidio/raw-go), so end users do **not** need Python, a virtual environment, or system libraries to run it.

## Features
- Supports common RAW formats (CR2, CR3, NEF, ARW, DNG, RAF, ORF, RW2, PEF, SRW, ...)
- Resize to a maximum dimension (keeps aspect ratio) at a configurable JPEG quality
- Preserves EXIF metadata (best effort, copied from the embedded thumbnail)
- Multithreaded processing (goroutine worker pool)
- Idempotent: skips images already present in the destination
- CLI arguments + optional JSON config file

## Build

Requires Go 1.24+ and a C toolchain (CGO).

```bash
git clone https://github.com/janistwigex/imageconverter.git
cd imageconverter
go build -o imageconverter .
```

Cross-compile example (Linux amd64):
```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o imageconverter .
```

## Run (no install needed for end users)
```bash
./imageconverter --input /mnt/nas/raws --output /mnt/nas/jpegs --size 2048 --quality 85 --workers 8
```

## Examples

### Basic conversion (default size 2048px, quality 85)
```bash
./imageconverter --input /mnt/nas/raws --output /mnt/nas/jpegs
```

### Resize to 1024px longest side, quality 90, 8 workers
```bash
./imageconverter -input /mnt/nas/raws -output /mnt/nas/jpegs -size 1024 -quality 90 -workers 8
```

### Keep original resolution (no resize), maximum quality
```bash
./imageconverter -input /mnt/nas/raws -output /mnt/nas/jpegs -size 0 -quality 100
```

### Process only Canon and Nikon files
```bash
./imageconverter -input /mnt/nas/raws -output /mnt/nas/jpegs -extensions cr2,cr3,nef
```

### Dry run (preview what would be converted)
```bash
./imageconverter -input /mnt/nas/raws -output /mnt/nas/jpegs -dry-run
```

### Force reprocess already-converted images
```bash
./imageconverter -input /mnt/nas/raws -output /mnt/nas/jpegs -force
```

### Use a config file
```bash
./imageconverter -config config.json
```

### Override a config value from the command line
```bash
./imageconverter -config config.json -quality 95 -workers 16
```

### Verbose per-file progress + log file
```bash
./imageconverter -input /mnt/nas/raws -output /mnt/nas/jpegs -verbose -log-file convert.log
```

## Config file
Edit `config.json` (auto-detected in CWD or next to the binary) and run:
```bash
./imageconverter -config config.json
```
CLI arguments override config values.

### Options
- `-input`: source root path
- `-output`: destination root path
- `-size`: max dimension in pixels (longest side). `0` or omitted = keep original size
- `-quality`: JPEG quality 1-100 (default 85)
- `-workers`: parallel workers (0 = CPU count)
- `-extensions`: comma-separated RAW extensions
- `-force`: reprocess existing outputs
- `-dry-run`: list files without converting
- `-verbose`: per-file progress
- `-config`: path to JSON config file
- `-log-file`: write log to file
