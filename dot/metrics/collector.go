// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

// GaugeMetrics interface allows the exportation of many gauge metrics
// the implementer could exports
type GaugeMetrics interface {
	CollectGauge() map[string]int64
}

// Collector struct controls the metrics and executes polling to extract the values
type Collector struct {
	ctx    context.Context
	gauges []GaugeMetrics
	wg     sync.WaitGroup
}

// NewCollector creates a new Collector
func NewCollector(ctx context.Context) *Collector {
	return &Collector{
		ctx:    ctx,
		wg:     sync.WaitGroup{},
		gauges: make([]GaugeMetrics, 0),
	}
}

// Start will start one goroutine to collect all the gauges registered and
// a separate goroutine to collect process metrics
func (c *Collector) Start() {
	ethmetrics.Enabled = true
	c.wg.Add(2)

	go c.startCollectProccessMetrics()
	go c.startCollectGauges()

	c.wg.Wait()
}

// AddGauge adds a GaugeMetrics implementer on gauges list
func (c *Collector) AddGauge(g GaugeMetrics) {
	c.gauges = append(c.gauges, g)
}

func (c *Collector) startCollectGauges() {
	//TODO: Should we better add individual RefreshInterval for each `GaugeMetrics`or `label inside the gauges map`?
	t := time.NewTicker(RefreshInterval)
	defer func() {
		t.Stop()
		c.wg.Done()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-t.C:
			for _, g := range c.gauges {
				m := g.CollectGauge()

				for label, value := range m {
					gauge := ethmetrics.GetOrRegisterGauge(label, nil)
					gauge.Update(value)
				}
			}
		}
	}
}

func (c *Collector) startCollectProccessMetrics() {
	//TODO: Should we better add individual RefreshInterval for each `GaugeMetrics`or `label inside the gauges map`?
	cpuStats := make([]*ethmetrics.CPUStats, 2)
	memStats := make([]*runtime.MemStats, 2)
	for i := 0; i < len(memStats); i++ {
		cpuStats[i] = new(ethmetrics.CPUStats)
		memStats[i] = new(runtime.MemStats)
	}

	// Define the various metrics to collect
	var (
		cpuSysLoad    = ethmetrics.GetOrRegisterGauge("system/cpu/sysload", ethmetrics.DefaultRegistry)
		cpuSysWait    = ethmetrics.GetOrRegisterGauge("system/cpu/syswait", ethmetrics.DefaultRegistry)
		cpuProcLoad   = ethmetrics.GetOrRegisterGauge("system/cpu/procload", ethmetrics.DefaultRegistry)
		cpuGoroutines = ethmetrics.GetOrRegisterGauge("system/cpu/goroutines", ethmetrics.DefaultRegistry)

		memPauses = ethmetrics.GetOrRegisterMeter("system/memory/pauses", ethmetrics.DefaultRegistry)
		memAlloc  = ethmetrics.GetOrRegisterMeter("system/memory/allocs", ethmetrics.DefaultRegistry)
		memFrees  = ethmetrics.GetOrRegisterMeter("system/memory/frees", ethmetrics.DefaultRegistry)
		memHeld   = ethmetrics.GetOrRegisterGauge("system/memory/held", ethmetrics.DefaultRegistry)
		memUsed   = ethmetrics.GetOrRegisterGauge("system/memory/used", ethmetrics.DefaultRegistry)
	)

	t := time.NewTicker(RefreshInterval)
	defer func() {
		t.Stop()
		c.wg.Done()
	}()

	for i := 1; ; i++ {
		select {
		case <-c.ctx.Done():
			return
		case <-t.C:
			location1 := i % 2
			location2 := (i - 1) % 2

			ethmetrics.ReadCPUStats(cpuStats[location1])
			cpuSysLoad.Update((cpuStats[location1].GlobalTime - cpuStats[location2].GlobalTime) / refreshFreq)
			cpuSysWait.Update((cpuStats[location1].GlobalWait - cpuStats[location2].GlobalWait) / refreshFreq)
			cpuProcLoad.Update((cpuStats[location1].LocalTime - cpuStats[location2].LocalTime) / refreshFreq)
			cpuGoroutines.Update(int64(runtime.NumGoroutine()))

			runtime.ReadMemStats(memStats[location1])
			memPauses.Mark(int64(memStats[location1].PauseTotalNs - memStats[location2].PauseTotalNs))
			memAlloc.Mark(int64(memStats[location1].Mallocs - memStats[location2].Mallocs))
			memFrees.Mark(int64(memStats[location1].Frees - memStats[location2].Frees))
			memHeld.Update(int64(memStats[location1].HeapSys - memStats[location1].HeapReleased))
			memUsed.Update(int64(memStats[location1].Alloc))
		}
	}
}
