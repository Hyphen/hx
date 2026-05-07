package timing

import (
	"fmt"
	"sync"
	"time"

	"github.com/Hyphen/cli/pkg/cprint"
)

type Phase struct {
	Label    string
	Duration time.Duration
}

type Recorder struct {
	started time.Time
	mu      sync.Mutex
	phases  []Phase
}

func NewRecorder() *Recorder {
	return &Recorder{started: time.Now()}
}

func (r *Recorder) Measure(label string, fn func() error) error {
	if r == nil {
		return fn()
	}

	started := time.Now()
	err := fn()
	r.Record(label, time.Since(started))
	return err
}

func (r *Recorder) Record(label string, duration time.Duration) {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.phases = append(r.phases, Phase{
		Label:    label,
		Duration: duration,
	})
	r.mu.Unlock()
}

func (r *Recorder) Print(printer *cprint.CPrinter, command string) {
	if r == nil || printer == nil {
		return
	}

	r.mu.Lock()
	phases := make([]Phase, len(r.phases))
	copy(phases, r.phases)
	total := time.Since(r.started)
	r.mu.Unlock()

	printer.PrintVerbose(fmt.Sprintf("%s timing:", command))
	for _, phase := range phases {
		printer.PrintVerbose(fmt.Sprintf("  %-20s %s", phase.Label, phase.Duration.Round(time.Millisecond)))
	}
	printer.PrintVerbose(fmt.Sprintf("  %-20s %s", "total", total.Round(time.Millisecond)))
}
