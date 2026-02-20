package dedupe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHashDeterministic(t *testing.T) {
	h := NewHasher(1024)
	p := filepath.Join(t.TempDir(), "x.bin")
	_ = os.WriteFile(p, []byte("abc123"), 0o644)
	a, _ := h.HashFile(context.Background(), p)
	b, _ := h.HashFile(context.Background(), p)
	if a != b {
		t.Fatal("hash not deterministic")
	}
}

func BenchmarkHashSmall(b *testing.B) {
	h := NewHasher(1024)
	p := filepath.Join(b.TempDir(), "s.bin")
	_ = os.WriteFile(p, []byte("abc123"), 0o644)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = h.HashFile(ctx, p)
	}
}
