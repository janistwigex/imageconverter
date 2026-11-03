# raw-go

A Go library and CLI tool to decode camera RAW images without system dependencies (CGO).

## 💡 Motivation

There are a couple of libraries to decode RAW images in Go, and even though they do the job well, they have one limitation that don't satisfy my needs: they depend on libraries to be installed on the system to be built and/or later be executed.

**raw-go** uses CGO to create a static implementation of [LibRaw](https://www.libraw.org) `0.22.1`, so you don't need to have `libraw` (or any of its sub-dependencies) installed to decode RAW images.

It also runs on native code (supports `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, `windows/arm64`), so it achieves the best performance possible.

> [!NOTE]
> LibRaw only _reads_ RAW files, so **raw-go** can decode them but it cannot encode/write them.

## ⬇️ Installation

### Library

This library can be installed using Go modules. To do that, run the following command in your project's root directory:

```bash
$ go get github.com/vegidio/raw-go
```

### CLI

The binaries are available for Windows, macOS, and Linux. Download the [latest release](https://github.com/vegidio/raw-go/releases) that matches your computer architecture and operating system.

## 🤖 Usage

### Library

This is a CGO library, so to use it, you _must_ enable CGO while building your application. You can do that by setting the `CGO_ENABLED` environment variable to `1`:

```bash
$ CGO_ENABLED=1 go build /path/to/your/app.go
```

Here are some examples of how to decode RAW images, extract their thumbnails and read their metadata using this library. These snippets don't have any error handling for the sake of simplicity, but you should always check for errors in production code.

#### Decoding

```go
rawFile, err := os.Open("/path/to/image.nef") // open the RAW file to be decoded
rawImage, err := raw.Decode(rawFile) // decode the RAW into an image.Image
```

You can also use the standard `image.Decode` for the formats that **raw-go** registers (Canon CR2 and Fujifilm RAF, the only ones with an unambiguous signature):

```go
import _ "github.com/vegidio/raw-go" // blank import to register the RAW decoders
rawFile, err := os.Open("/path/to/image.cr2") // open the RAW file to be decoded
rawImage, _, err := image.Decode(rawFile) // decode the image
```

By default `raw.Decode` applies the camera's as-shot white balance and auto-brightness, producing a natural rendering close to the camera's own JPEG. Note that the in-camera JPEG bakes in a proprietary picture-style tone curve that a RAW developer cannot reproduce exactly, so the two will never be pixel-identical. You can tune the result with options:

```go
rawImage, err := raw.Decode(rawFile,
    raw.WithWhiteBalance(raw.WhiteBalanceCamera), // camera (default), auto, or none
    raw.WithBrightness(1.5),                      // brightness multiplier (1.0 = neutral)
    raw.WithAutoBright(false),                    // disable auto-bright to keep the as-shot exposure
)
```

#### Thumbnail

```go
rawFile, err := os.Open("/path/to/image.nef") // open the RAW file
thumbnail, err := raw.ExtractThumbnail(rawFile) // extract the embedded preview as JPEG bytes
err = os.WriteFile("/path/to/thumb.jpg", thumbnail, 0o644) // save the thumbnail
```

#### Metadata

```go
rawFile, err := os.Open("/path/to/image.nef") // open the RAW file
meta, err := raw.ReadMetadata(rawFile) // read the camera metadata
fmt.Printf("%s %s - %dx%d\n", meta.Make, meta.Model, meta.Width, meta.Height)
```

### CLI

If you want to decode a RAW image, run the following command:

```bash
$ raw decode /path/to/image.nef /path/to/image.png
```

The `decode` command accepts `--white-balance` (`camera`/`auto`/`none`, default `camera`), `--brightness` (multiplier, default `1.0`), and `--no-auto-bright` (preserve the as-shot exposure) to tune the rendering.

---

To extract the embedded thumbnail of a RAW image, run the following command:

```bash
$ raw thumb /path/to/image.nef /path/to/thumb.jpg
```

---

To print the camera metadata of a RAW image, run the following command:

```bash
$ raw info /path/to/image.nef
```

For the full list of parameters, type `raw decode --help` in the terminal.

## 📷 Supported formats

LibRaw deliberately **does not** rely on a file-extension list — it detects RAW files by inspecting their _content_, so `raw.Decode` works regardless of the file's extension ([details](https://www.libraw.org/node/1421)). The table below lists the extensions conventionally associated with the formats LibRaw `0.22.1` can open (based on digiKam's `rawfiles.h`); a given `.tif`/`.tiff` or `.dng` may or may not be RAW depending on the camera that produced it.

| Manufacturer                  | Extensions                                |
|-------------------------------|-------------------------------------------|
| Canon                         | `.crw` `.cr2` `.cr3`                       |
| Nikon                         | `.nef` `.nrw`                             |
| Sony                          | `.arw` `.srf` `.sr2`                       |
| Fujifilm                      | `.raf`                                     |
| Olympus / OM System           | `.orf`                                     |
| Panasonic / Leica             | `.rw2` `.raw` `.rwl`                       |
| Pentax / Ricoh                | `.pef` `.ptx` `.dng`                       |
| Samsung                       | `.srw`                                     |
| Sigma (Foveon)                | `.x3f`                                     |
| Hasselblad / Imacon           | `.3fr` `.fff`                             |
| Phase One                     | `.iiq` `.cap` `.eip`                       |
| Kodak                         | `.dcr` `.kdc` `.k25` `.dcs` `.dc2`         |
| Leaf                          | `.mos`                                     |
| Mamiya                        | `.mef`                                     |
| Minolta                       | `.mrw` `.mdc`                             |
| Epson                         | `.erf`                                     |
| Casio                         | `.bay`                                     |
| Logitech                      | `.pxn`                                     |
| GoPro                         | `.gpr`                                     |
| Adobe / generic & smartphones | `.dng`                                     |
| Other / misc                  | `.bmq` `.cs1` `.cine` `.ia` `.kc2` `.qtk` `.rdc` `.sti` |

## 💣 Troubleshooting

### I cannot build my app after importing this library

If you cannot build your app after importing **raw-go**, it is probably because you didn't set the `CGO_ENABLED` environment variable to `1`.

You must either set a global environment variable with `export CGO_ENABLED=1` or set it in the command line when building your app with `CGO_ENABLED=1 go build /path/to/your/app.go`.

### "App Is Damaged/Blocked..." (Windows & macOS only)

For a couple of years now, Microsoft and Apple have required developers to join their "Developer Program" to gain the pretentious status of an _identified developer_ 😛.

Translating to non-BS language, this means that if you’re not registered with them (i.e., paying the fee), you can’t freely distribute Windows or macOS software. Apps from unidentified developers will display a message saying the app is damaged or blocked and can’t be opened.

To bypass this, open the Terminal and run one of the commands below (depending on your operating system), replacing `<path-to-app>` with the correct path to where you’ve installed the app:

- Windows: `Unblock-File -Path <path-to-app>`
- macOS: `xattr -d com.apple.quarantine <path-to-app>`

## 📝 License

**raw-go** is released under the Apache 2.0 License. See [LICENSE](LICENSE) for details.

## 👨🏾‍💻 Author

Vinicius Egidio ([vinicius.io](http://vinicius.io))
