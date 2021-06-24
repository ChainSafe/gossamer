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
	"runtime"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

const (
	// Refresh is the refresh time for publishing metrics.
	Refresh     = time.Second * 10
	refreshFreq = int64(Refresh / time.Second)
)

// CollectProcessMetrics periodically collects various metrics about the running process.
func CollectProcessMetrics() {
	// Create the various data collectors
	cpuStats := make([]*metrics.CPUStats, 2)
	memStats := make([]*runtime.MemStats, 2)
	for i := 0; i < len(memStats); i++ {
		cpuStats[i] = new(metrics.CPUStats)
		memStats[i] = new(runtime.MemStats)
	}

	// Define the various metrics to collect
	var (
		cpuSysLoad    = metrics.GetOrRegisterGauge("system/cpu/sysload", metrics.DefaultRegistry)
		cpuSysWait    = metrics.GetOrRegisterGauge("system/cpu/syswait", metrics.DefaultRegistry)
		cpuProcLoad   = metrics.GetOrRegisterGauge("system/cpu/procload", metrics.DefaultRegistry)
		cpuGoroutines = metrics.GetOrRegisterGauge("system/cpu/goroutines", metrics.DefaultRegistry)

		memPauses = metrics.GetOrRegisterMeter("system/memory/pauses", metrics.DefaultRegistry)
		memAlloc  = metrics.GetOrRegisterMeter("system/memory/allocs", metrics.DefaultRegistry)
		memFrees  = metrics.GetOrRegisterMeter("system/memory/frees", metrics.DefaultRegistry)
		memHeld   = metrics.GetOrRegisterGauge("system/memory/held", metrics.DefaultRegistry)
		memUsed   = metrics.GetOrRegisterGauge("system/memory/used", metrics.DefaultRegistry)
	)

	// Iterate loading the different stats and updating the meters
	for i := 1; ; i++ {
		location1 := i % 2
		location2 := (i - 1) % 2

		metrics.ReadCPUStats(cpuStats[location1])
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

		time.Sleep(Refresh)
	}
}
