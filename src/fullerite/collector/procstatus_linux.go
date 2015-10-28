// +build linux

package collector

import (
	"fullerite/metric"

	"strconv"
	"strings"

	"github.com/prometheus/procfs"
)

// Collect produces some random test metrics.
func (ps ProcStatus) Collect() {
	for _, m := range ps.procStatusMetrics() {
		ps.Channel() <- m
	}
}

func procStatusPoint(name string, value float64, dimensions map[string]string) (m metric.Metric) {
	m = metric.New(name)
	m.Value = value
	m.AddDimension("collector", "ProcStatus")
	m.AddDimensions(dimensions)
	return m
}

func (ps ProcStatus) getMetrics(proc procfs.Proc, cmdOutput []string) []metric.Metric {
	stat, err := proc.NewStat()
	if err != nil {
		ps.log.Warn("Error getting stats: ", err)
		return nil
	}

	pid := strconv.Itoa(stat.PID)
	processName := stat.Comm
	if len(ps.processName) > 0 {
		processName = ps.processName
	}

	dim := map[string]string{
		"processName": processName,
		"pid":         pid,
	}

	ret := []metric.Metric{
		procStatusPoint("VirtualMemory", float64(stat.VirtualMemory()), dim),
		procStatusPoint("ResidentMemory", float64(stat.ResidentMemory()), dim),
		procStatusPoint("CPUTime", float64(stat.CPUTime()), dim),
	}

	if len(cmdOutput) > 0 {
		generatedDimensions := ps.extractDimensions(cmdOutput[0])
		metric.AddToAll(&ret, generatedDimensions)
	}

	return ret
}

func (ps ProcStatus) procStatusMetrics() []metric.Metric {
	procs, err := procfs.AllProcs()
	if err != nil {
		ps.log.Warn("Error getting processes: ", err)
		return nil
	}

	ret := []metric.Metric{}

	for _, proc := range procs {
		cmd, err := proc.CmdLine()
		if err != nil {
			ps.log.Warn("Error getting command line: ", err)
			continue
		}

		if len(ps.processName) == 0 || len(cmd) > 0 && strings.Contains(cmd[0], ps.processName) {
			ret = append(ret, ps.getMetrics(proc, cmd)...)
		}
	}

	return ret
}

func (ps ProcStatus) extractDimensions(cmd string) map[string]string {
	ret := map[string]string{}

	for dimension, procRegex := range ps.compiledRegex {
		subMatch := procRegex.FindStringSubmatch(cmd)
		if len(subMatch) > 1 {
			ret[dimension] = subMatch[1]
		}
	}

	return ret
}
