<div align="center">
  <img src="assets/logo.png" alt="MediaCrunch Logo" width="160"/>

  # MediaCrunch

  [![Discord](https://img.shields.io/badge/Discord-Join%20Community-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.gg/nnkfW83n)
  [![Go](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
  [![License](https://img.shields.io/badge/License-Non--Commercial%20Copyleft-orange?style=for-the-badge)](LICENSE)

</div>

MediaCrunch is a local CPU-only image transcoding daemon with WebP + AVIF support, durable SQLite history, smart skip policy, hash dedupe, metrics, benchmarks, and optional pprof.

## Codecs
- **webp**: faster, good default.
- **avif**: smaller output, slower encode.

## Dependencies (Ubuntu 22.04+)
```bash
sudo apt install build-essential pkg-config sqlite3 libwebp-dev libjpeg-dev libpng-dev libheif-dev
```

## Build / Test / CI
```bash
make build
make test
make ci
```

## Transcode CLI
```bash
./bin/mediacrunchd transcode --in ./testdata/sample.jpg --out ./build/out.webp --codec webp --q 82
./bin/mediacrunchd transcode --in ./testdata/sample.jpg --out ./build/out.avif --codec avif --q 60
```

## Daemon CLI
```bash
./bin/mediacrunchd daemon \
  --input ./testdata/in \
  --output ./build/out \
  --db ./build/mediacrunch.db \
  --codec webp \
  --workers 4 \
  --quality 82 \
  --write-mode atomic \
  --on-success keep \
  --metrics-addr 127.0.0.1:9090
```

## Bench Harness
```bash
./bin/mediacrunchd bench --input ./testdata/bench --out ./build/bench --codec webp --q 82 --runs 3
```
Writes JSON summary to `./build/bench/report.json` with throughput, bytes, ratio, and duration.

## Smart Skip + Dedupe
A job is skipped when any condition is true:
1. Output exists, is valid for codec, and is fresh vs input (`output_fresh`).
2. Store has matching prior success (same path/codec/quality) and output remains valid (`store_success`).
3. SHA-256 hash dedupe finds an existing valid output (`deduped_hash`).

## Metrics
`GET /metrics` exposes:
- `mediacrunch_jobs_enqueued_total`
- `mediacrunch_jobs_rejected_total`
- `mediacrunch_jobs_processed_total`
- `mediacrunch_jobs_failed_total`
- `mediacrunch_jobs_skipped_total`
- `mediacrunch_bytes_in_total`
- `mediacrunch_bytes_out_total`

## CPU Profiling (optional)
Profiling is off by default. Enable with `--pprof-addr` on daemon or bench.
```bash
./bin/mediacrunchd daemon ... --pprof-addr 127.0.0.1:6060
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=15
```

## Troubleshooting
- Watcher is non-recursive: put inputs directly in `--input`.
- If `database is locked`, ensure only one daemon uses the DB.
- If AVIF build fails, verify `libheif-dev` is installed and visible to `pkg-config`.
