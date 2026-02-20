package image

/*
#cgo CFLAGS: -I${SRCDIR}/../../../native/imgcodec
#cgo CXXFLAGS: -std=c++17 -I${SRCDIR}/../../../native/imgcodec
#cgo pkg-config: libwebp libjpeg libpng
#include <stdlib.h>
#include "imgcodec.h"
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

type Transcoder struct{}

func NewTranscoder() *Transcoder {
	return &Transcoder{}
}

func (t *Transcoder) TranscodeToWebP(ctx context.Context, inPath, outPath string, quality int) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled before transcode: %w", ctx.Err())
	default:
	}

	cin := C.CString(inPath)
	defer C.free(unsafe.Pointer(cin))
	cout := C.CString(outPath)
	defer C.free(unsafe.Pointer(cout))

	const errBufLen = 1024
	errBuf := make([]byte, errBufLen)

	status := C.mc_transcode_webp(cin, cout, C.int(quality), (*C.char)(unsafe.Pointer(&errBuf[0])), C.int(len(errBuf)))
	if status != 0 {
		msg := C.GoString((*C.char)(unsafe.Pointer(&errBuf[0])))
		if msg == "" {
			msg = "unknown native error"
		}
		return fmt.Errorf("native transcoder error (%d): %s", int(status), msg)
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled after transcode: %w", ctx.Err())
	default:
	}

	return nil
}
