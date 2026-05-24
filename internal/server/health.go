package server

import (
	"math"
	"runtime"
	"sync"
	"time"
)

var (
	startMu   sync.Once
	startTime time.Time
)

const mb float64 = 1.0 * 1024 * 1024

// InitStartTime records the server start time.
// Must be called once when the server starts listening.
func InitStartTime() {
	startMu.Do(func() {
		startTime = time.Now()
	})
}

type HealthStats struct {
	Uptime               int64   `json:"uptime"`
	AllocatedMemory      float64 `json:"allocatedMemory"`
	TotalAllocatedMemory float64 `json:"totalAllocatedMemory"`
	Goroutines           int     `json:"goroutines"`
	GCCycles             uint32  `json:"completedGCCycles"`
	NumberOfCPUs         int     `json:"cpus"`
	HeapSys              float64 `json:"maxHeapUsage"`
	HeapAllocated        float64 `json:"heapInUse"`
	ObjectsInUse         uint64  `json:"objectsInUse"`
	OSMemoryObtained     float64 `json:"OSMemoryObtained"`
}

func GetHealthStats() *HealthStats {
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)

	return &HealthStats{
		Uptime:               GetUptime(),
		AllocatedMemory:      toMegaBytes(mem.Alloc),
		TotalAllocatedMemory: toMegaBytes(mem.TotalAlloc),
		Goroutines:           runtime.NumGoroutine(),
		NumberOfCPUs:         runtime.NumCPU(),
		GCCycles:             mem.NumGC,
		HeapSys:              toMegaBytes(mem.HeapSys),
		HeapAllocated:        toMegaBytes(mem.HeapAlloc),
		ObjectsInUse:         mem.Mallocs - mem.Frees,
		OSMemoryObtained:     toMegaBytes(mem.Sys),
	}
}

func GetUptime() int64 {
	return time.Now().Unix() - startTime.Unix()
}

func toMegaBytes(bytes uint64) float64 {
	return toFixed(float64(bytes)/mb, 2)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
