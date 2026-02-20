package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	enqueued  atomic.Int64
	rejected  atomic.Int64
	processed atomic.Int64
	failed    atomic.Int64
	skipped   atomic.Int64
	bytesIn   atomic.Int64
	bytesOut  atomic.Int64
}

func New() *Metrics { return &Metrics{} }

func (m *Metrics) IncEnqueued()  { m.enqueued.Add(1) }
func (m *Metrics) IncRejected()  { m.rejected.Add(1) }
func (m *Metrics) IncProcessed() { m.processed.Add(1) }
func (m *Metrics) IncFailed()    { m.failed.Add(1) }
func (m *Metrics) IncSkipped()   { m.skipped.Add(1) }
func (m *Metrics) AddBytes(in, out int64) {
	m.bytesIn.Add(in)
	m.bytesOut.Add(out)
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w,
			"mediacrunch_jobs_enqueued_total %d\nmediacrunch_jobs_rejected_total %d\nmediacrunch_jobs_processed_total %d\nmediacrunch_jobs_failed_total %d\nmediacrunch_jobs_skipped_total %d\nmediacrunch_bytes_in_total %d\nmediacrunch_bytes_out_total %d\n",
			m.enqueued.Load(), m.rejected.Load(), m.processed.Load(), m.failed.Load(), m.skipped.Load(), m.bytesIn.Load(), m.bytesOut.Load(),
		)
	})
}
