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
	"time"

	ethmetrics "github.com/ethereum/go-ethereum/metrics"
)

type GaugeMetrics interface {
	CollectGauge() map[string]int64
}

type Collector struct {
	ctx    context.Context
	gauges []GaugeMetrics
}

func NewCollector(ctx context.Context) *Collector {
	return &Collector{
		ctx:    ctx,
		gauges: make([]GaugeMetrics, 0),
	}
}

func (c *Collector) Start() {
	go c.startCollectProccessMetrics()
	go c.startCollectGauges()
}

func (c *Collector) Stop() {
	for _, g := range c.gauges {
		m := g.CollectGauge()

		for label := range m {
			ethmetrics.Unregister(label)
		}
	}
}

func (c *Collector) AddGauge(g GaugeMetrics) {
	c.gauges = append(c.gauges, g)
}

func (c *Collector) startCollectGauges() {
	ethmetrics.Enabled = true

	t := time.NewTicker(Refresh)
	defer t.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-t.C:
			for _, g := range c.gauges {
				m := g.CollectGauge()

				for label, value := range m {
					pooltx := ethmetrics.GetOrRegisterGauge(label, nil)
					pooltx.Update(value)
				}
			}
		}
	}
}

func (c *Collector) startCollectProccessMetrics() {
	ethmetrics.Enabled = true

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

	t := time.NewTicker(Refresh)
	defer t.Stop()

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
