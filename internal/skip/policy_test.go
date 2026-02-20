package skip

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mediacrunch/internal/dedupe"
	"mediacrunch/internal/jobs"
	storesqlite "mediacrunch/internal/store/sqlite"
)

func BenchmarkSkipDecision(b *testing.B) {
	db := filepath.Join(b.TempDir(), "m.db")
	st, _ := storesqlite.New(db)
	_ = st.Init(context.Background())
	h := dedupe.NewHasher(1024)
	idx := dedupe.NewSQLiteIndex(db)
	p := NewPolicy(st, idx, h, jobs.CodecWebP, 80)
	in := filepath.Join(b.TempDir(), "in.jpg")
	out := filepath.Join(b.TempDir(), "out.webp")
	_ = os.WriteFile(in, []byte("abc"), 0o644)
	_ = os.WriteFile(out, []byte("RIFFxxxxWEBPxxxx"), 0o644)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = p.ShouldSkip(context.Background(), in, out)
	}
}
