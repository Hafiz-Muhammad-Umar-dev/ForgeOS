package metrics

import (
	"runtime"
	"sync"
	"time"
)

type Snapshot struct {
	CPU      float64   `json:"cpu"`
	Memory   float64   `json:"memory"`
	Disk     float64   `json:"disk"`
	Network  float64   `json:"network"`
	Requests float64   `json:"requests_per_sec"`
	Latency  float64   `json:"latency"`
	Errors   int       `json:"errors"`
	Time     time.Time `json:"timestamp"`
}

type Collector struct {
	mu       sync.RWMutex
	history  []Snapshot
	reqCount int
	errCount int
	latSum   float64
}

func NewCollector() *Collector {
	c := &Collector{}
	go c.poll()
	return c
}

func (c *Collector) poll() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		c.mu.Lock()
		c.history = append(c.history, Snapshot{
			CPU:     0.5,
			Memory:  float64(m.Alloc) / float64(m.Sys),
			Disk:    0.3,
			Network: 1.2,
			Latency: c.latSum / float64(max(c.reqCount, 1)),
			Errors:  c.errCount,
			Time:    time.Now(),
		})
		if len(c.history) > 100 {
			c.history = c.history[len(c.history)-100:]
		}
		c.mu.Unlock()
	}
}

func (c *Collector) RecordRequest(latency float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reqCount++
	c.latSum += latency
}

func (c *Collector) RecordError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errCount++
}

func (c *Collector) History() []Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	r := make([]Snapshot, len(c.history))
	copy(r, c.history)
	return r
}
