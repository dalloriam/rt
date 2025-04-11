package rt

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type scheduler struct {
	log  *slog.Logger
	quit chan struct{}
	rt   *Runtime
	wg   sync.WaitGroup
}

func newScheduler(rt *Runtime, log *slog.Logger) *scheduler {
	if log == nil {
		log = slog.Default()
	}
	return &scheduler{
		log:  log.WithGroup("sched"),
		quit: make(chan struct{}),
		rt:   rt,
	}
}

func (s *scheduler) runTaskSchedule(process ProcessFn, opts ProcessSpawnOptions, interval time.Duration) {
	for {
		select {
		case <-s.quit:
			s.wg.Done()
			return
		case <-time.After(interval):
			s.rt.ProcSpawn(context.Background(), process, opts)
		}
	}
}

func (s *scheduler) Schedule(process ProcessFn, opts ProcessSpawnOptions, interval time.Duration) {
	s.wg.Add(1)
	go s.runTaskSchedule(process, opts, interval)
}

func (s *scheduler) Close() {
	close(s.quit)
	s.wg.Wait()
}
