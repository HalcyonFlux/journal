package server

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/vaitekunas/lentele"
	"github.com/vaitekunas/unixsock"
)

// ManagementConsole handles commands received over the unix socket
type ManagementConsole interface {

	// CmdStatistics displays various statistics
	CmdStatistics(unixsock.Args) *unixsock.Response

	// CmdLogsList list all available logfiles and their archives
	CmdLogsList(unixsock.Args) *unixsock.Response

	// CmdRemoteAdd adds a remote backend
	CmdRemoteAdd(unixsock.Args) *unixsock.Response

	// CmdRemoteList lists all active remote backends
	CmdRemoteList(unixsock.Args) *unixsock.Response

	// CmdRemoteRemove removes a remote backend
	CmdRemoteRemove(unixsock.Args) *unixsock.Response

	// CmdTokensAdd adds a new token for a service/instance
	CmdTokensAdd(unixsock.Args) *unixsock.Response

	// CmdTokensListInstances lists all permitted instances of a service
	CmdTokensListInstances(unixsock.Args) *unixsock.Response

	// CmdTokensListServices lists all permitted services
	CmdTokensListServices(unixsock.Args) *unixsock.Response

	// CmdTokensRemoveInstance removes the token of a service/instance
	CmdTokensRemoveInstance(unixsock.Args) *unixsock.Response

	// CmdTokensRemoveService removes the token of all instances of a service
	CmdTokensRemoveService(unixsock.Args) *unixsock.Response

	// Execute is the executor of management console commands
	Execute(string, unixsock.Args) *unixsock.Response
}

// NewConsole creates a new management console for the log server
func NewConsole(server *LogServer) ManagementConsole {

	return &managementConsole{
		logserver: server,
	}
}

// managementConsole handles commands received over the unix socket
type managementConsole struct {
	banner    string
	logserver *LogServer
}

// Execute is the executor of management console commands
func (m *managementConsole) Execute(cmd string, args unixsock.Args) *unixsock.Response {

	fmt.Println(console(bold(strings.ToLower(cmd))))

	switch strings.ToLower(cmd) {

	case "statistics":
		return m.CmdStatistics(args)

	case "tokens.add":
		return m.CmdTokensAdd(args)

	case "tokens.remove.instance":
		return m.CmdTokensRemoveInstance(args)

	case "tokens.remove.service":
		return m.CmdTokensRemoveService(args)

	case "tokens.list.instances":
		return m.CmdTokensListInstances(args)

	case "tokens.list.services":
		return m.CmdTokensListServices(args)

	case "logs.list":
		return m.CmdLogsList(args)

	case "remote.add":
		return m.CmdRemoteAdd(args)

	case "remote.remove":
		return m.CmdRemoteRemove(args)

	case "remote.list":
		return m.CmdRemoteList(args)

	default:
		return &unixsock.Response{
			Status: "failure",
			Error:  fmt.Errorf("Execute: unknown command '%s'", cmd).Error(),
		}
	}

}

// arg is a helper struct used to for slices of required arguments
type arg struct {
	Name string
	Kind reflect.Kind
}

// validArguments verifies that all the required arguments have been provided
func validArguments(args unixsock.Args, required []arg) bool {
	for _, f := range required {
		x, ok := args[f.Name]
		if !ok {
			return false
		}

		if reflect.TypeOf(x).Kind() != f.Kind {
			return false
		}
	}
	return true
}

var respMissingArgs = &unixsock.Response{
	Status: "failure",
	Error:  fmt.Errorf("Missing/invalid parameters").Error(),
}

// CmdStatistics displays various log-related statistics
func (m *managementConsole) CmdStatistics(args unixsock.Args) *unixsock.Response {
	m.logserver.Lock()
	defer m.logserver.Unlock()

	type aggregate struct {
		service   string
		instances int
		volume    int64
		logs      int64
		share     float64
	}

	hourlyLogs := [24]int64{}
	hourlyVolume := [24]int64{}
	hourlyVolumeShare := make([]float64, 24)
	hours := make([]interface{}, 24)

	// Aggregate data
	var totalLogVolume int64
	serviceAggroMap := map[string]*aggregate{}
	serviceNames := []string{}
	for _, stats := range m.logserver.stats {

		service := stats.Service
		_, _, plogs, pbytes := parsedSums(stats.LogsParsed, stats.LogsParsedBytes)

		serviceAggro, ok := serviceAggroMap[service]
		if !ok {
			serviceNames = append(serviceNames, service)
			serviceAggro = &aggregate{service: service}
			serviceAggroMap[service] = serviceAggro
		}

		// Hourly statistics
		for i := 0; i < 24; i++ {
			hourlyLogs[i] += stats.LogsParsed[i]
			hourlyVolume[i] += stats.LogsParsedBytes[i]
		}

		serviceAggro.instances++
		serviceAggro.logs += plogs
		serviceAggro.volume += pbytes

		totalLogVolume += pbytes
	}

	// Calculate shares
	shares := make([]float64, len(serviceNames))
	i := 0
	for _, stsum := range serviceAggroMap {
		stsum.share = float64(stsum.volume) / float64(totalLogVolume)
		shares[i] = stsum.share
		i++
	}

	// Service table
	serviceTable := lentele.New("Service", "Instances", "Logs sent", "Volume share")
	shareSort := &floatSorter{floats: shares}
	sort.Sort(shareSort)
	idx := shareSort.GetIndexes()
	for i := range idx {
		service := serviceNames[i]
		mp := serviceAggroMap[service]
		plogStr, pbyteStr := prettyParsedSums(mp.logs, mp.volume)
		serviceTable.AddRow("").Insert(service, mp.instances, fmt.Sprintf("%s (%s)", plogStr, pbyteStr), fmt.Sprintf("%6.2f%%", mp.share*100))
	}

	// Hourly table
	hourlyTable := lentele.New("Hour", "Logs sent", "Volume", "Volume share")
	for i := 0; i < 24; i++ {

		var hour string
		if i < 10 {
			hour = fmt.Sprintf("0%d", i)
		} else {
			hour = fmt.Sprintf("%d", i)
		}
		hours[i] = hour

		plogsStr, pbytesStr := prettyParsedSums(hourlyLogs[i], hourlyVolume[i])
		share := float64(hourlyVolume[i]) / float64(totalLogVolume)

		row := hourlyTable.AddRow("")
		hourlyVolumeShare[i] = share
		row.Insert(hour, plogsStr, pbytesStr, fmt.Sprintf("%6.2f%%", share*100))
	}

	// Print tables and barchart
	buf := bytes.NewBuffer([]byte{})
	serviceTable.Render(buf, false, true, true, lentele.LoadTemplate("classic"))
	buf.WriteString("\n")
	barchart(buf, hours, hourlyVolumeShare, "▧", color.New(color.FgHiGreen), 10, 1, true)
	buf.WriteString("\n")
	hourlyTable.Render(buf, false, true, true, lentele.LoadTemplate("classic"))

	// Successful op
	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("journald statistics:\n%s", buf.String())),
	}

}

// CmdTokensAdd adds a new token for a service/instance
func (m *managementConsole) CmdTokensAdd(args unixsock.Args) *unixsock.Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
		arg{"instance", reflect.String},
	}

	// TODO: match service instance names to a [a-z][0-9]-_. regex

	// Validate arguments
	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service/instance
	service := args["service"].(string)
	instance := args["instance"].(string)
	token, err := m.logserver.AddToken(service, instance)
	if err != nil {
		return &unixsock.Response{
			Status: "failure",
			Error:  fmt.Errorf("Could not add token: %s", err.Error()).Error(),
		}
	}

	// Prepare table
	table := lentele.New("Service", "Instance", "Token")
	table.AddRow("").Insert(service, instance, token).Modify(bold, "Token")
	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	// Successful op
	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("added token for '%s':\n%s", bold(getCleanKey(service, instance)), buf.String())),
	}

}

// CmdTokensRemoveInstance removes the token of a service/instance
func (m *managementConsole) CmdTokensRemoveInstance(args unixsock.Args) *unixsock.Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
		arg{"instance", reflect.String},
	}

	// Validate arguments
	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service/instance
	service := args["service"].(string)
	instance := args["instance"].(string)
	if err := m.logserver.RemoveToken(service, instance); err != nil {
		return &unixsock.Response{
			Status: "failure",
			Error:  fmt.Errorf("Could not remove token: %s", err.Error()).Error(),
		}
	}

	// Successful op
	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("removed token for '%s'\n", bold(getCleanKey(service, instance)))),
	}

}

// CmdTokensRemoveService removes the token of all instances of a service
func (m *managementConsole) CmdTokensRemoveService(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdTokensListInstances lists all permitted instances of a service
func (m *managementConsole) CmdTokensListInstances(args unixsock.Args) *unixsock.Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
	}

	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service
	service := strings.ToLower(args["service"].(string))

	// Prepare table
	table := lentele.New("Instance", "Token", "Last known IP", "Logs sent")

	m.logserver.Lock()
	for key, token := range m.logserver.tokens {
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == service {
			ip := m.logserver.stats[key].LastIP
			plogs := m.logserver.stats[key].LogsParsed
			pbytes := m.logserver.stats[key].LogsParsedBytes
			plogsStr, pbytesStr, _, _ := parsedSums(plogs, pbytes)

			table.AddRow("").Insert(parts[1], fmt.Sprintf("%s...", token[0:10]), ip, fmt.Sprintf("%s (%s)", plogsStr, pbytesStr))
		}
	}
	m.logserver.Unlock()

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("available instances for service %s:\n%s", bold(service), buf.String())),
	}
}

// CmdTokensListServices lists all permitted services
func (m *managementConsole) CmdTokensListServices(args unixsock.Args) *unixsock.Response {

	// Prepare statistics
	serviceNames := []string{}
	services := map[string][2]int{}
	for key := range m.logserver.tokens {
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			continue
		}
		if _, ok := services[parts[0]]; !ok {
			serviceNames = append(serviceNames, parts[0])
			services[parts[0]] = [2]int{}
		}
		counts := services[parts[0]]
		counts[0]++
	}
	sort.Strings(serviceNames)

	busy := func(v interface{}) interface{} {
		return color.New(color.FgRed).Sprint(v)
	}

	// Prepare table
	table := lentele.New("", "Service", "Instances", "Last log entry", "Log entries parsed")
	for _, name := range serviceNames {
		service := services[name]
		now := time.Now().Format("2006-01-02 15:04")

		table.AddRow("").Insert("●", name, service[0], now, service[1]).Modify(busy, "")

	}

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: buf.String(),
	}
}

// CmdLogsList list all available logfiles and their archives
func (m *managementConsole) CmdLogsList(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteAdd adds a remote backend
func (m *managementConsole) CmdRemoteAdd(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteRemove removes a remote backend
func (m *managementConsole) CmdRemoteRemove(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteList lists all active remote backends
func (m *managementConsole) CmdRemoteList(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}
