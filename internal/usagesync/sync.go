package usagesync

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Options configure a [Run].
type Options struct {
	Source     Source
	Targets    []Target
	WindowDays int
	// Problems are targets that could not be resolved, keyed by address;
	// they are carried into the snapshot's errors.
	Problems map[string]string
	// Now is the end of the window. Zero means the current time.
	Now time.Time
	// Concurrency bounds the lookups in flight. Zero means 8.
	Concurrency int
}

// Run measures every target and returns the snapshot. A failure for one
// target is recorded under its address and does not stop the others. The
// error is non-nil only when the context ended before the run did.
func Run(ctx context.Context, opts Options) (Snapshot, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	win := Window{Start: now.AddDate(0, 0, -opts.WindowDays), End: now}

	snap := Snapshot{
		Version: SchemaVersion,
		Synced: Meta{
			Provider:    opts.Source.Provider(),
			WindowDays:  opts.WindowDays,
			GeneratedAt: now,
		},
		ResourceUsage: map[string]map[string]any{},
		Series:        map[string]map[string]map[string]float64{},
		Errors:        map[string]string{},
	}
	for addr, msg := range opts.Problems {
		snap.Errors[addr] = msg
	}

	workers := opts.Concurrency
	if workers <= 0 {
		workers = 8
	}
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		jobs = make(chan Target)
	)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				res, err := collect(ctx, opts.Source, t, win)
				mu.Lock()
				switch {
				case err != nil:
					snap.Errors[t.Address] = err.Error()
				default:
					values := make(map[string]any, len(res.Values))
					for k, v := range res.Values {
						values[k] = v
					}
					snap.ResourceUsage[t.Address] = values
					if len(res.Series) > 0 {
						snap.Series[t.Address] = res.Series
					}
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for _, t := range opts.Targets {
		select {
		case jobs <- t:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()
	return snap, ctx.Err()
}

// collect calls the source and turns a panic in it into an error for that
// target, so one bad response cannot take the whole run down.
func collect(ctx context.Context, s Source, t Target, w Window) (res Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	return s.Collect(ctx, t, w)
}
