# imageconverter

Convert camera RAW images to JPEG while preserving the input folder structure.

## Features
- Supports common RAW formats (CR2, CR3, NEF, ARW, DNG, RAF, ORF, RW2, PEF, SRW, ...)
- Resize to a maximum dimension (keeps aspect ratio) at a configurable JPEG quality
- Preserves EXIF metadata (best effort, from embedded thumbnail)
- Multithreaded processing
- Idempotent: skips images already present in the destination
- CLI arguments + optional JSON config file

## Installation

### 1. Clone the repository
```bash
git clone https://github.com/janistwigex/imageconverter.git
cd imageconverter
```

### 2. (Recommended) Create a virtual environment
```bash
python3 -m venv .venv
source .venv/bin/activate        # Linux / macOS
# .venv\\Scripts\\activate       # Windows
```

### 3. Install dependencies
```bash
pip install -r requirements.txt
```

Or install manually:
```bash
pip install rawpy Pillow numpy
```
## Usage
```bash
python3 imageconverter.py --input /mnt/nas/raws --output /mnt/nas/jpegs \
    --size 2048 --quality 85 --workers 8
```

## Examples

### Basic conversion (default size 2048px, quality 85)
```bash
python3 imageconverter.py --input /mnt/nas/raws --output /mnt/nas/jpegs
```

### Resize to 1024px longest side, quality 90, 8 workers
```bash
python3 imageconverter.py -i /mnt/nas/raws -o /mnt/nas/jpegs --size 1024 --quality 90 --workers 8
```

### Keep original resolution (no resize), maximum quality
```bash
python3 imageconverter.py -i /mnt/nas/raws -o /mnt/nas/jpegs --size 0 --quality 100
```

### Process only Canon and Nikon files
```bash
python3 imageconverter.py -i /mnt/nas/raws -o /mnt/nas/jpegs --extensions cr2,cr3,nef
```

### Dry run (preview what would be converted)
```bash
python3 imageconverter.py -i /mnt/nas/raws -o /mnt/nas/jpegs --dry-run
```

### Force reprocess already-converted images
```bash
python3 imageconverter.py -i /mnt/nas/raws -o /mnt/nas/jpegs --force
```

### Use a config file
```bash
python3 imageconverter.py --config config.json
```

### Override a config value from the command line
```bash
python3 imageconverter.py --config config.json --quality 95 --workers 16
```

### Verbose per-file progress + log file
```bash
python3 imageconverter.py -i /mnt/nas/raws -o /mnt/nas/jpegs -v --log-file convert.log
```

### Windows UNC-style paths (when mounted) example
```bash
python3 imageconverter.py -i "Z:\\photos\\raws" -o "Z:\\photos\\jpegs" --size 2048
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
