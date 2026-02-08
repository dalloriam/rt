package sched

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/dalloriam/rt/api"
	privApi "github.com/dalloriam/rt/internal/api"
)

type Scheduler struct {
	log  *slog.Logger
	quit chan struct{}
	rt   privApi.Runtime
	wg   sync.WaitGroup
}

func New(rt privApi.Runtime, log *slog.Logger) *Scheduler {
	if log == nil {
		log = slog.Default()
	}
	return &Scheduler{
		log:  log.WithGroup("sched"),
		quit: make(chan struct{}),
		rt:   rt,
	}
}

func (s *Scheduler) runTaskSchedule(process api.ProcessFn, opts api.SpawnOptions, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.rt.Proc().Spawn(context.Background(), process, opts)
		}
	}
}

func (s *Scheduler) Schedule(process api.ProcessFn, opts api.SpawnOptions, interval time.Duration) {
	if interval <= 0 {
		s.log.Error("invalid schedule interval", "interval", interval)
		return
	}

	s.wg.Add(1)
	go s.runTaskSchedule(process, opts, interval)
}

func (s *Scheduler) Close() {
	close(s.quit)
	s.wg.Wait()
}
