package worker

import (
	"log"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type Stats struct {
	MemStats  *mem.VirtualMemoryStat
	DiskStats *disk.UsageStat
	CPUStats  []cpu.TimesStat
	LoadStats *load.AvgStat
	TaskCount int
}

func GetStats() *Stats {
	return &Stats{
		MemStats:  GetMemoryInfo(),
		DiskStats: GetDiskInfo(),
		CPUStats:  GetCPUStats(),
		LoadStats: GetLoadAvg(),
	}
}

func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.Total / 1024
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.Available / 1024
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.Used / 1024
}

func (s *Stats) MemUsedPercent() uint64 {
	if s.MemStats.Total == 0 {
		return 0
	}
	return uint64(s.MemStats.UsedPercent)
}

func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.Total
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

// CPUUsage returns the CPU usage as a percentage:
// https://stackoverflow.com/questions/23367857/accurate-calculation-of-cpu-usage-given-in-percentage-in-linux
func (s *Stats) CPUUsage() float64 {
	if len(s.CPUStats) == 0 {
		return 0.0
	}

	// Use the first (combined) CPU stat
	cpuStat := s.CPUStats[0]

	idle := cpuStat.Idle + cpuStat.Iowait
	nonIdle := cpuStat.User + cpuStat.Nice + cpuStat.System +
		cpuStat.Irq + cpuStat.Softirq + cpuStat.Steal
	total := idle + nonIdle

	if total == 0 {
		return 0.0
	}
	return (total - idle) / total
}

func GetMemoryInfo() *mem.VirtualMemoryStat {
	memstats, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Error reading memory info: %v", err)
		return &mem.VirtualMemoryStat{}
	}
	return memstats
}

func GetDiskInfo() *disk.UsageStat {
	diskstats, err := disk.Usage("/")
	if err != nil {
		log.Printf("Error reading disk info: %v", err)
		return &disk.UsageStat{}
	}
	return diskstats
}

func GetCPUStats() []cpu.TimesStat {
	stats, err := cpu.Times(false) // false = combined stats
	if err != nil {
		log.Printf("Error reading CPU stats: %v", err)
		return []cpu.TimesStat{}
	}
	return stats
}

func GetLoadAvg() *load.AvgStat {
	loadavg, err := load.Avg()
	if err != nil {
		log.Printf("Error reading load average: %v", err)
		return &load.AvgStat{}
	}
	return loadavg
}
