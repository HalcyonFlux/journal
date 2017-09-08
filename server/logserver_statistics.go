package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/vaitekunas/journal/logrpc"

	context "golang.org/x/net/context"
)

// GatherStatistics saves log-related statistics
func (l *LogServer) GatherStatistics(service, instance, key, ip string, logEntry *logrpc.LogEntry) {
	l.Lock()
	defer l.Unlock()

	now := time.Now()

	if _, ok := l.stats[key]; !ok {
		l.stats[key] = &Statistic{
			Service:         service,
			Instance:        instance,
			LogsParsed:      [24]int64{},
			LogsParsedBytes: [24]int64{},
		}
	}

	jsoned, err := json.Marshal(logEntry.GetEntry())
	if err != nil {
		jsoned = []byte{}
	}

	stats := l.stats[key]
	stats.LogsParsed[now.Hour()]++
	stats.LogsParsedBytes[now.Hour()] += int64(len(jsoned))
	stats.LastIP = ip
	stats.LastActive = now
}

// AggregateStatistics contains aggregated logging statistics
type AggregateStatistics struct {
	service   string
	instances int
	volume    int64
	logs      int64
	share     float64
}

// AggregateServiceStatistics aggregates statistics
func (l *LogServer) AggregateServiceStatistics() (totalVolume int64, services []*AggregateStatistics, hourly [24][2]int64) {
	l.Lock()
	defer l.Unlock()

	// Aggregate data
	var totalLogVolume int64
	serviceAggroMap := map[string]*AggregateStatistics{}
	serviceNames := []string{}
	hourly = [24][2]int64{}
	for _, stats := range l.stats {

		service := stats.Service
		_, _, plogs, pbytes := parsedSums(stats.LogsParsed, stats.LogsParsedBytes)

		serviceAggro, ok := serviceAggroMap[service]
		if !ok {
			serviceNames = append(serviceNames, service)
			serviceAggro = &AggregateStatistics{service: service}
			serviceAggroMap[service] = serviceAggro
		}

		for i := 0; i <= 23; i++ {
			hourly[i][0] += stats.LogsParsed[i]
			hourly[i][1] += stats.LogsParsedBytes[i]
		}

		serviceAggro.instances++
		serviceAggro.logs += plogs
		serviceAggro.volume += pbytes

		totalLogVolume += pbytes
	}

	// Calculate shares
	shares := make([]float64, len(serviceNames))
	for i, name := range serviceNames {
		stsum := serviceAggroMap[name]
		stsum.share = float64(stsum.volume) / float64(totalLogVolume)
		shares[i] = stsum.share
	}

	// Sort by share
	shareSort := &floatSorter{floats: shares}
	sort.Sort(shareSort)
	aggro := make([]*AggregateStatistics, len(shares))
	for i := range shareSort.GetIndexes() {
		aggro[i] = serviceAggroMap[serviceNames[i]]
	}

	return totalLogVolume, aggro, hourly
}

// periodicallyDumpStats periodically dumps statistics to file
func (l *LogServer) periodicallyDumpStats(ctx context.Context, period time.Duration) {
Loop:
	for {
		select {
		case <-time.After(period):
			l.dumpStatsToFile()
		case <-ctx.Done():
			break Loop
		}
	}
}

// dumpStatsToFile dumps all the statistics into file
func (l *LogServer) dumpStatsToFile() error {
	l.Lock()
	defer l.Unlock()

	// Make sure file exists
	if err := fileExists(l.statsPath); err != nil {
		return fmt.Errorf("dumpStatsToFile: could not create statistics database: %s", err.Error())
	}

	// JSON statistics
	jsoned, errJSON := json.Marshal(l.stats)
	if errJSON != nil {
		return fmt.Errorf("dumpStatsToFile: could not marshal statistics to json: %s", errJSON.Error())
	}

	// Open file for reading
	f, err := os.OpenFile(l.statsPath, os.O_WRONLY, 600)
	if err != nil {
		return fmt.Errorf("dumpStatsToFile: could not open statistics file for writing: %s", err.Error())
	}

	// Write stats
	if _, err := f.Write(jsoned); err != nil {
		defer f.Close()
		return fmt.Errorf("dumpStatsToFile: could not dump stats: %s", err.Error())
	}

	return f.Close()
}

// loadStatisticsFromDisk loads server statistics from file
func (l *LogServer) loadStatisticsFromDisk() error {
	l.Lock()
	defer l.Unlock()

	// Make sure file exists
	if err := fileExists(l.statsPath); err != nil {
		return fmt.Errorf("loadStatisticsFromDisk: could not create statistics database: %s", err.Error())
	}

	// Open file for reading
	f, err := os.OpenFile(l.statsPath, os.O_RDONLY, 600)
	if err != nil {
		return fmt.Errorf("loadStatisticsFromDisk: could not open statistics file for reading: %s", err.Error())
	}
	defer f.Close()

	// Read json-encoded statistics
	jsoned, err := ioutil.ReadFile(l.statsPath)
	if err != nil {
		return fmt.Errorf("loadStatisticsFromDisk: could not read file: %s", err.Error())
	}
	if len(jsoned) == 0 {
		return nil
	}

	// Unmarshal json-encoded statistics
	if err := json.Unmarshal(jsoned, &l.stats); err != nil {
		return fmt.Errorf("loadStatisticsFromDisk: could not unmarshal statistics: %s", err.Error())
	}

	return nil
}
