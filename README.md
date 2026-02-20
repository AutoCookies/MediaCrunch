# MediaCrunch

MediaCrunch is a CPU-only image transcoding daemon. It converts JPEG/PNG inputs into WebP using a Go orchestrator and a native C++ codec worker via cgo.

## Architecture

```text
+--------------------------+       +--------------------------+
| cmd/mediacrunchd (Go)    | ----> | internal/app             |
+--------------------------+       +-----------+--------------+
                                                |
                        +-----------------------+-----------------------+
                        |                                               |
                        v                                               v
              +-------------------+                         +-------------------+
              | daemon + watcher  | ---- paths -----------> | orchestrator      |
              | fsnotify          |                         | worker pool       |
              +-------------------+                         | bounded queue     |
                                                            | skip policy       |
                                                            +---------+---------+
                                                                      |
                                                         +------------+-------------+
                                                         |                          |
                                                         v                          v
                                              +-------------------+      +-------------------+
                                              | sqlite job store  |      | output atomic     |
                                              | history + stats   |      | temp + fsync + mv |
                                              +-------------------+      +-------------------+
                                                                      |
                                                                      v
                                                           +-------------------+
                                                           | native imgcodec   |
                                                           | jpeg/png -> webp  |
                                                           +-------------------+
```

## Dependencies (Ubuntu 22.04+)

```bash
sudo apt install build-essential pkg-config libwebp-dev libjpeg-dev libpng-dev
```

## Build

```bash
make build
```

Binary:

```bash
./bin/mediacrunchd
```

## Test

```bash
make test
```

## CI-equivalent local run

```bash
make ci
```

## CLI Usage

### One-shot transcode

```bash
./bin/mediacrunchd transcode --in ./testdata/sample.jpg --out ./build/out.webp --q 82
```

### Daemon mode

```bash
./bin/mediacrunchd daemon \
  --input ./testdata/in \
  --output ./build/out \
  --db ./build/mediacrunch.db \
  --workers 4 \
  --quality 82 \
  --write-mode atomic \
  --on-success keep \
  --metrics-addr 127.0.0.1:9090
```

## Daemon Flags

- `--input`: watched input directory (non-recursive in Phase 2).
- `--output`: destination directory for generated `.webp` files.
- `--db`: SQLite database path.
- `--workers`: worker count.
- `--queue-size`: bounded queue size (default `256`).
- `--quality`: webp quality `[1..100]`.
- `--write-mode`: must be `atomic`.
- `--on-success`: must be `keep`.
- `--metrics-addr`: HTTP bind address for `/metrics`.

## Metrics

Endpoint:

```text
GET /metrics
```

Exposed counters:

- `mediacrunch_jobs_enqueued_total`
- `mediacrunch_jobs_rejected_total`
- `mediacrunch_jobs_processed_total`
- `mediacrunch_jobs_failed_total`
- `mediacrunch_jobs_skipped_total`
- `mediacrunch_bytes_in_total`
- `mediacrunch_bytes_out_total`

## Local end-to-end run

```bash
mkdir -p ./testdata/in ./build/out ./build
./testdata/generate_sample.sh
cp ./testdata/sample.jpg ./testdata/in/input.jpg
./bin/mediacrunchd daemon --input ./testdata/in --output ./build/out --db ./build/mediacrunch.db --workers 4 --quality 82 --write-mode atomic --on-success keep --metrics-addr 127.0.0.1:9090
```

In another shell:

```bash
curl -s http://127.0.0.1:9090/metrics
```

Verify output exists:

```bash
ls ./build/out/*.webp
```

## Troubleshooting

- **No files processed**: watcher is non-recursive; place files directly under `--input`.
- **Permission errors**: ensure daemon user can read input, write output, and write DB directory.
- **`database is locked`**: avoid multiple daemon instances sharing one DB path.
- **Missing native libs**: install `libwebp-dev`, `libjpeg-dev`, `libpng-dev`.
