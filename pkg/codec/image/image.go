package image

/*
#cgo CFLAGS: -I${SRCDIR}/../../../native/imgcodec
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../../native/imgcodec
#cgo pkg-config: libwebp libjpeg libpng libheif
#include <stdlib.h>
#include "imgcodec.h"
#include "avifcodec.h"
*/
import "C"

import (
	"context"
	"fmt"
	"os"
	"time"
	"unsafe"

	"mediacrunch/internal/jobs"
)

type Result struct {
	BytesIn  int64
	BytesOut int64
	Duration time.Duration
	Codec    jobs.ImageCodec
}

type Transcoder struct{}

func NewTranscoder() *Transcoder { return &Transcoder{} }

func (t *Transcoder) Transcode(ctx context.Context, codec jobs.ImageCodec, inPath, outPath string, quality int) (Result, error) {
	start := time.Now()
	select {
	case <-ctx.Done():
		return Result{}, fmt.Errorf("context cancelled before transcode: %w", ctx.Err())
	default:
	}
	cin := C.CString(inPath)
	defer C.free(unsafe.Pointer(cin))
	cout := C.CString(outPath)
	defer C.free(unsafe.Pointer(cout))

	errBuf := make([]byte, 1024)
	var status C.int
	switch codec {
	case jobs.CodecWebP:
		status = C.mc_transcode_webp(cin, cout, C.int(quality), (*C.char)(unsafe.Pointer(&errBuf[0])), C.int(len(errBuf)))
	case jobs.CodecAVIF:
		status = C.mc_transcode_avif(cin, cout, C.int(quality), (*C.char)(unsafe.Pointer(&errBuf[0])), C.int(len(errBuf)))
	default:
		return Result{}, fmt.Errorf("unsupported codec %q", codec)
	}
	if status != 0 {
		msg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		if msg == "" {
			msg = "unknown native error"
		}
		return Result{}, fmt.Errorf("native transcoder error (%d): %s", int(status), msg)
	}
	return Result{BytesIn: fileSize(inPath), BytesOut: fileSize(outPath), Duration: time.Since(start), Codec: codec}, nil
}

func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return fi.Size()
}
