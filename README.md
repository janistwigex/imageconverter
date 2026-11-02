# imageconverter

Convert camera RAW images to JPEG while preserving the input folder structure.

## Features
- Supports common RAW formats (CR2, CR3, NEF, ARW, DNG, RAF, ORF, RW2, PEF, SRW, ...)
- Resize to a maximum dimension (keeps aspect ratio) at a configurable JPEG quality
- Preserves EXIF metadata (best effort, from embedded thumbnail)
- Multithreaded processing
- Idempotent: skips images already present in the destination
- CLI arguments + optional JSON config file

## Install dependencies
```bash
pip install -r requirements.txt
```

## Usage
```bash
python3 imageconverter.py --input /mnt/nas/raws --output /mnt/nas/jpegs \
    --size 2048 --quality 85 --workers 8
```

### Options
- `--input` / `-i`: source root path
- `--output` / `-o`: destination root path
- `--size`: max dimension in pixels (longest side). `0` or omitted = keep original size
- `--quality`: JPEG quality 1-100 (default 85)
- `--workers`: parallel workers (0 = CPU count)
- `--extensions`: comma-separated RAW extensions
- `--force`: reprocess existing outputs
- `--dry-run`: list files without converting
- `--verbose` / `-v`: per-file progress
- `--config`: path to JSON config file
- `--log-file`: write log to file

### Config file
Edit `config.json` (auto-detected in CWD or script dir) and run:
```bash
python3 imageconverter.py --config config.json
```
CLI arguments override config values.
