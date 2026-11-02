#!/usr/bin/env python3
"""Convert camera RAW images to JPEG while preserving folder structure.

Usage:
    python3 imageconverter.py --input /mnt/nas/raws --output /mnt/nas/jpegs \\
        --size 2048 --quality 85 --workers 8

Settings can also be provided via a JSON config file (see config.json).
Command-line arguments override config file values.
"""

import argparse
import json
import logging
import os
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

try:
    import rawpy
except ImportError as e:  # pragma: no cover
    print("rawpy is required. Install with: pip install rawpy", file=sys.stderr)
    raise

try:
    from PIL import Image
except ImportError as e:  # pragma: no cover
    print("Pillow is required. Install with: pip install Pillow", file=sys.stderr)
    raise

RAW_EXTENSIONS = {
    ".cr2", ".cr3", ".nef", ".arw", ".dng", ".raf", ".orf", ".rw2",
    ".pef", ".srw", ".raw", ".crw", ".nrw", ".sr2", ".srf", ".mrw",
    ".erf", ".kdc", ".dcr", ".3fr", ".fff", ".iiq", ".rwl", ".x3f",
}


def load_config(path):
    if not path:
        return {}
    p = Path(path)
    if not p.is_file():
        return {}
    with p.open("r", encoding="utf-8") as f:
        return json.load(f)


def find_config():
    candidates = [Path.cwd() / "config.json", Path(__file__).resolve().parent / "config.json"]
    for c in candidates:
        if c.is_file():
            return str(c)
    return None


def resize_image(img, max_dim):
    if not max_dim:
        return img
    w, h = img.size
    if max(w, h) <= max_dim:
        return img
    img = img.copy()
    img.thumbnail((max_dim, max_dim), Image.LANCZOS)
    return img


def extract_exif(raw):
    """Best-effort EXIF bytes from the RAW's embedded thumbnail JPEG."""
    try:
        thumb = raw.extract_thumb()
    except Exception:
        return None
    fmt = getattr(thumb, "format", None)
    # rawpy.ThumbFormat.JPEG
    is_jpeg = False
    try:
        is_jpeg = (fmt == rawpy.ThumbFormat.JPEG) or str(fmt).endswith("JPEG")
    except Exception:
        is_jpeg = "JPEG" in str(fmt).upper()
    if not is_jpeg:
        return None
    data = getattr(thumb, "data", None)
    if not data:
        return None
    try:
        import io
        with Image.open(io.BytesIO(data)) as timg:
            return timg.info.get("exif")
    except Exception:
        return None


def convert_one(src, dst, max_dim, quality, force):
    """Convert a single RAW file. Returns (status, src, message)."""
    src = str(src)
    dst = str(dst)
    if (not force) and os.path.isfile(dst) and os.path.getsize(dst) > 0:
        return ("skipped", src, "already exists")
    try:
        os.makedirs(os.path.dirname(dst) or ".", exist_ok=True)
    except OSError as e:
        return ("error", src, f"mkdir failed: {e}")

    tmp = dst + ".tmp"
    try:
        with rawpy.imread(src) as raw:
            rgb = raw.postprocess(use_camera_wb=True, output_bps=8)
            img = Image.fromarray(rgb)
            img = resize_image(img, max_dim)
            exif_bytes = extract_exif(raw)

            save_kwargs = {"format": "JPEG", "quality": int(quality), "optimize": True}
            if exif_bytes:
                save_kwargs["exif"] = exif_bytes
            img.save(tmp, **save_kwargs)
        os.replace(tmp, dst)
        return ("ok", src, "")
    except Exception as e:
        try:
            if os.path.exists(tmp):
                os.remove(tmp)
        except OSError:
            pass
        return ("error", src, str(e))


def collect_tasks(input_root, output_root, extensions):
    tasks = []
    input_root = Path(input_root)
    output_root = Path(output_root)
    out_resolved = os.path.realpath(output_root)
    for dirpath, dirs, files in os.walk(input_root):
        # avoid descending into the output tree when it lives inside input
        dirs[:] = [d for d in dirs if os.path.realpath(os.path.join(dirpath, d)) != out_resolved
                   and not os.path.realpath(os.path.join(dirpath, d)).startswith(out_resolved + os.sep)]
        for name in files:
            ext = os.path.splitext(name)[1].lower()
            if ext not in extensions:
                continue
            src = Path(dirpath) / name
            rel = src.relative_to(input_root)
            # replace extension with .jpg
            dst_rel = rel.with_suffix(".jpg")
            dst = output_root / dst_rel
            tasks.append((str(src), str(dst)))
    return tasks


def parse_args():
    parser = argparse.ArgumentParser(description="Convert camera RAW images to JPEG.")
    parser.add_argument("--config", help="Path to JSON config file")
    parser.add_argument("--input", "-i", help="Input (source) root path")
    parser.add_argument("--output", "-o", help="Output (destination) root path")
    parser.add_argument("--size", type=int, help="Max dimension in pixels (longest side). 0 = no resize")
    parser.add_argument("--quality", type=int, help="JPEG quality 1-100")
    parser.add_argument("--workers", type=int, help="Number of parallel workers (0 = CPU count)")
    parser.add_argument("--extensions", help="Comma-separated list of RAW extensions to process")
    parser.add_argument("--force", action="store_true", help="Reprocess even if output exists")
    parser.add_argument("--dry-run", action="store_true", help="List files without converting")
    parser.add_argument("--verbose", "-v", action="store_true", help="Print per-file progress")
    parser.add_argument("--log-file", help="Write log to file")
    return parser.parse_args()


def main():
    args = parse_args()

    # resolve config
    config_path = args.config
    if not config_path:
        config_path = find_config()
    config = load_config(config_path)

    def cfg(key, cli_val, default=None):
        if cli_val is not None:
            return cli_val
        if key in config and config[key] is not None:
            return config[key]
        return default

    input_root = cfg("input", args.input)
    output_root = cfg("output", args.output)
    if not input_root or not output_root:
        print("ERROR: --input and --output are required (or set in config).", file=sys.stderr)
        sys.exit(2)

    size_raw = cfg("size", args.size, 0)
    size = int(size_raw) if size_raw else 0
    max_dim = size if size and size > 0 else None

    quality = int(cfg("quality", args.quality, 85))
    if not (1 <= quality <= 100):
        print("ERROR: quality must be 1-100", file=sys.stderr)
        sys.exit(2)

    workers_raw = cfg("workers", args.workers, 0)
    workers = int(workers_raw) if workers_raw else 0
    if workers <= 0:
        workers = os.cpu_count() or 4

    if args.extensions:
        exts = [e.strip().lower() for e in args.extensions.split(",") if e.strip()]
        extensions = {e if e.startswith(".") else "." + e for e in exts}
    elif "extensions" in config and config["extensions"]:
        exts = [str(e).strip().lower() for e in config["extensions"]]
        extensions = {e if e.startswith(".") else "." + e for e in exts}
    else:
        extensions = RAW_EXTENSIONS

    force = bool(cfg("force", args.force, False) or args.force)
    dry_run = bool(cfg("dry_run", args.dry_run, False) or args.dry_run)
    verbose = args.verbose

    # logging
    log_level = logging.DEBUG if verbose else logging.INFO
    handlers = [logging.StreamHandler(sys.stdout)]
    log_file = cfg("log_file", args.log_file, None)
    if log_file:
        handlers.append(logging.FileHandler(log_file, encoding="utf-8"))
    logging.basicConfig(level=log_level, format="%(asctime)s %(levelname)s %(message)s",
                        handlers=handlers, force=True)
    log = logging.getLogger("imageconverter")
    # Keep third-party (Pillow/rawpy) debug noise out of our output.
    for noisy in ("PIL", "PIL.TiffImagePlugin", "rawpy"):
        logging.getLogger(noisy).setLevel(logging.WARNING)

    log.info("Input : %s", input_root)
    log.info("Output: %s", output_root)
    log.info("Size  : %s", ("max " + str(max_dim) + "px") if max_dim else "original")
    log.info("Quality: %d", quality)
    log.info("Workers: %d", workers)
    log.info("Extensions: %s", ", ".join(sorted(extensions)))

    if not os.path.isdir(input_root):
        print(f"ERROR: input path not found or not a directory: {input_root}", file=sys.stderr)
        sys.exit(2)

    tasks = collect_tasks(input_root, output_root, extensions)
    log.info("Found %d RAW files to consider.", len(tasks))

    if dry_run:
        for src, dst in tasks:
            status = "SKIP" if (os.path.isfile(dst) and os.path.getsize(dst) > 0 and not force) else "CONVERT"
            print(f"{status}: {src} -> {dst}")
        return

    if not tasks:
        log.info("Nothing to do.")
        return

    ok = skipped = errors = 0
    start = time.time()
    total = len(tasks)

    def progress(i):
        if verbose or i % 50 == 0 or i == total:
            log.info("Progress: %d/%d (ok=%d skipped=%d errors=%d)", i, total, ok, skipped, errors)

    with ThreadPoolExecutor(max_workers=workers) as pool:
        future_map = {pool.submit(convert_one, src, dst, max_dim, quality, force): (src, dst)
                      for src, dst in tasks}
        done = 0
        for fut in as_completed(future_map):
            status, src, msg = fut.result()
            done += 1
            if status == "ok":
                ok += 1
                if verbose:
                    log.info("OK   : %s", src)
            elif status == "skipped":
                skipped += 1
                if verbose:
                    log.info("SKIP : %s (%s)", src, msg)
            else:
                errors += 1
                log.error("ERR  : %s -> %s", src, msg)
            progress(done)

    elapsed = time.time() - start
    log.info("Done in %.1fs. ok=%d skipped=%d errors=%d total=%d", elapsed, ok, skipped, errors, total)


if __name__ == "__main__":
    main()
