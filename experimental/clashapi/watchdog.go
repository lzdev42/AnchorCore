package clashapi

import (
	"os"
	"sync/atomic"
	"time"

	"github.com/sagernet/sing-box/log"
)

type Watchdog struct {
	lastFeedTime atomic.Int64
	logger       log.Logger
	onTimeout    func()
}

func NewWatchdog(logger log.Logger) *Watchdog {
	return &Watchdog{
		logger: logger,
		onTimeout: func() {
			logger.Fatal("Watchdog timeout: UI disconnected, self-destructing...")
			os.Exit(0)
		},
	}
}

func (w *Watchdog) Feed() {
	w.lastFeedTime.Store(time.Now().UnixNano())
}

func (w *Watchdog) Start() {
	w.Feed()
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			if time.Since(time.Unix(0, w.lastFeedTime.Load())) > 3*time.Second {
				w.onTimeout()
			}
		}
	}()
}
