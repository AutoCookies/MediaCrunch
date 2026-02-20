package dedupe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

type Hasher interface {
	HashFile(ctx context.Context, path string) (string, error)
}

type FileHasher struct {
	LargeFileThreshold int64
}

func NewHasher(threshold int64) *FileHasher {
	if threshold <= 0 {
		threshold = 256 << 20
	}
	return &FileHasher{LargeFileThreshold: threshold}
}

func (h *FileHasher) HashFile(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	hash := sha256.New()
	buf := make([]byte, 128*1024)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		n, rerr := f.Read(buf)
		if n > 0 {
			if _, err := hash.Write(buf[:n]); err != nil {
				return "", err
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return "", rerr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (h *FileHasher) Fingerprint(ctx context.Context, path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	first := make([]byte, 64*1024)
	n1, _ := io.ReadFull(f, first)
	last := make([]byte, 64*1024)
	if st.Size() > int64(len(last)) {
		if _, err := f.Seek(st.Size()-int64(len(last)), io.SeekStart); err == nil {
			n2, _ := io.ReadFull(f, last)
			last = last[:n2]
		}
	}
	s := sha256.Sum256([]byte(fmt.Sprintf("%d|%d|%x|%x", st.Size(), st.ModTime().UnixNano(), first[:n1], last)))
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	return hex.EncodeToString(s[:]), nil
}
