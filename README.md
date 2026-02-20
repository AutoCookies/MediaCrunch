# MediaCrunch Phase 1

MediaCrunch is a CPU-only image transcoding MVP. Phase 1 provides a production-grade CLI that converts JPEG/PNG images to WebP by orchestrating work in Go and executing decode/encode in native C++ via cgo. The sample input JPEG is stored as base64 (`testdata/sample.jpg.b64`) and generated deterministically to avoid committing binary blobs.

## Architecture

```text
+--------------------------+
| cmd/mediacrunchd (Go)    |
| - CLI parsing            |
| - command dispatch       |
+------------+-------------+
             |
             v
+--------------------------+
| internal/app + config    |
| - validation             |
| - orchestration          |
| - error propagation      |
+------------+-------------+
             |
             v (cgo C ABI)
+--------------------------+
| native/imgcodec (C++)    |
| - JPEG decode (libjpeg)  |
| - PNG decode (libpng)    |
| - WebP encode (libwebp)  |
+--------------------------+
```

## Install dependencies (Ubuntu 22.04+)

```bash
sudo apt install build-essential pkg-config libwebp-dev
```

> Note: Development headers for JPEG and PNG are also required for native decoding.

```bash
sudo apt install libjpeg-dev libpng-dev
```

## Build

```bash
make build
```

Binary output:

```bash
./bin/mediacrunchd
```

## Test

```bash
make test
```

`make build` and `make test` both ensure `./testdata/sample.jpg` is generated from `./testdata/sample.jpg.b64`.

## CLI usage

```bash
./bin/mediacrunchd transcode --in ./testdata/sample.jpg --out ./build/out.webp --q 82
```

## Troubleshooting

- **`pkg-config` cannot find libraries**: ensure `pkg-config`, `libwebp-dev`, `libjpeg-dev`, and `libpng-dev` are installed.
- **`unsupported input extension`**: only `.jpg`, `.jpeg`, and `.png` are accepted.
- **`invalid quality`**: quality must be between `1` and `100`.
- **Build fails in CI/local**: run `go vet ./...`, `go test ./...`, and `make build` to reproduce exact CI checks.
