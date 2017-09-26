package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"time"

	"github.com/vaitekunas/journal/logrpc"

	context "golang.org/x/net/context"
)

// GatherStatistics saves log-related statistics
func (l *logServer) GatherStatistics(service, instance, key, ip string, logEntry *logrpc.LogEntry) {
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

// AggregateStatistics contains aggregated statistics
type AggregateStatistics struct {
	Service   string
	Instances int
	Volume    int64
	Logs      int64
	Share     float64
}

// GetStatistics returns LogServer's statistics
func (l *logServer) GetStatistics() map[string]*Statistic {
	l.Lock()
	defer l.Unlock()

	copyStats := map[string]*Statistic{}
	for key, stats := range l.stats {

		logsParsed := [24]int64{}
		logsParsedBytes := [24]int64{}
		copy(logsParsed[:24], stats.LogsParsed[:24])
		copy(logsParsedBytes[:24], stats.LogsParsedBytes[:24])

		copyStats[key] = &Statistic{
			Service:         stats.Service,
			Instance:        stats.Instance,
			LogsParsed:      logsParsed,
			LogsParsedBytes: logsParsedBytes,
			LastIP:          stats.LastIP,
			LastActive:      stats.LastActive,
		}
	}

	return copyStats
}

// AggregateServiceStatistics aggregates statistics
func (l *logServer) AggregateServiceStatistics() (totalVolume int64, services []*AggregateStatistics, hourly [24][2]int64) {
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
			serviceAggro = &AggregateStatistics{Service: service}
			serviceAggroMap[service] = serviceAggro
		}

		for i := 0; i <= 23; i++ {
			hourly[i][0] += stats.LogsParsed[i]
			hourly[i][1] += stats.LogsParsedBytes[i]
		}

		serviceAggro.Instances++
		serviceAggro.Logs += plogs
		serviceAggro.Volume += pbytes

		totalLogVolume += pbytes
	}

	// Calculate shares
	shares := make([]float64, len(serviceNames))
	for i, name := range serviceNames {
		stsum := serviceAggroMap[name]
		stsum.Share = float64(stsum.Volume) / float64(totalLogVolume)
		shares[i] = stsum.Share
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
func (l *logServer) periodicallyDumpStats(ctx context.Context, period time.Duration) {
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
func (l *logServer) dumpStatsToFile() error {
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

	// Write stats
	if err := ioutil.WriteFile(l.statsPath, jsoned, 0600); err != nil {
		return fmt.Errorf("dumpStatsToFile: could not dump stats: %s", err.Error())
	}

	return nil
}

// loadStatisticsFromDisk loads server statistics from file
func (l *logServer) loadStatisticsFromDisk() error {
	l.Lock()
	defer l.Unlock()

	// Make sure file exists
	if err := fileExists(l.statsPath); err != nil {
		return fmt.Errorf("loadStatisticsFromDisk: could not create statistics database: %s", err.Error())
	}

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
