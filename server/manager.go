package server

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/vaitekunas/journal"
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

	// Get aggregated statistics
	totalLogVolume, aggro, hourly := m.logserver.AggregateServiceStatistics()

	// Service table
	serviceTable := lentele.New("Service", "Instances", "Logs sent", "Volume share")
	for _, service := range aggro {
		plogStr, pbyteStr := prettyParsedSums(service.logs, service.volume)
		serviceTable.AddRow("").Insert(service.service, service.instances, fmt.Sprintf("%s (%s)", plogStr, pbyteStr), fmt.Sprintf("%6.2f%%", service.share*100))
	}

	// Hourly table
	hourlyTable := lentele.New("Hour", "Logs sent", "Volume", "Volume share")
	hourlyVolumeShare := make([]float64, 24)
	hours := make([]interface{}, 24)
	for i, stats := range hourly {

		var hour string
		if i < 10 {
			hour = fmt.Sprintf("0%d", i)
		} else {
			hour = fmt.Sprintf("%d", i)
		}
		hours[i] = hour

		plogsStr, pbytesStr := prettyParsedSums(stats[0], stats[1])
		share := float64(stats[1]) / float64(totalLogVolume)
		hourlyVolumeShare[i] = share
		if stats[0] > 0 {
			row := hourlyTable.AddRow("")
			row.Insert(hour, plogsStr, pbytesStr, fmt.Sprintf("%6.2f%%", share*100))
		}
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
	if err := m.logserver.RemoveToken(service, instance, true); err != nil {
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

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
	}

	// Validate arguments
	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service/instance
	service := args["service"].(string)
	if err := m.logserver.RemoveTokens(service); err != nil {
		return &unixsock.Response{
			Status: "failure",
			Error:  fmt.Errorf("Could not remove tokens for the service '%s': %s", service, err.Error()).Error(),
		}
	}

	// Successful op
	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("removed all tokens for service '%s'\n", bold(service))),
	}

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

	// Lock tokens
	m.logserver.Lock()
	defer m.logserver.Unlock()

	// Identify service
	service := strings.ToLower(args["service"].(string))

	// Prepare table
	table := lentele.New("Instance", "Token", "Last known IP", "Logs sent")

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

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("available instances for service %s:\n%s", bold(service), buf.String())),
	}
}

// CmdTokensListServices lists all permitted services
func (m *managementConsole) CmdTokensListServices(args unixsock.Args) *unixsock.Response {

	// Get aggregated statistics
	_, aggro, _ := m.logserver.AggregateServiceStatistics()

	// Service table
	table := lentele.New("Service", "Instances", "Logs sent", "Volume share")
	for _, service := range aggro {
		plogStr, pbyteStr := prettyParsedSums(service.logs, service.volume)
		table.AddRow("").Insert(service.service, service.instances, fmt.Sprintf("%s (%s)", plogStr, pbyteStr), fmt.Sprintf("%6.2f%%", service.share*100))
	}

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("available services:\n%s", buf.String())),
	}
}

// CmdLogsList list all available logfiles and their archives
func (m *managementConsole) CmdLogsList(args unixsock.Args) *unixsock.Response {

	tail := -1

	if show, ok := args["show"]; ok {
		if showInt, okInt := show.(float64); okInt && showInt > 0 {
			tail = int(showInt)
		}
	}

	logs, err := m.logserver.Logfiles()
	if err != nil {
		return &unixsock.Response{
			Status: unixsock.STATUS_FAIL,
			Error:  err.Error(),
		}
	}

	names := make([]string, len(logs))
	i := 0
	for name := range logs {
		names[i] = name
		i++
	}

	sort.Strings(names)
	if tail > 0 && len(names) >= tail {
		names = names[len(names)-tail:]
	}

	table := lentele.New("Logfile", "Size")
	for _, name := range names {
		if name == "" {
			continue
		}
		table.AddRow("").Insert(name, logs[name])
	}

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: console(fmt.Sprintf("available logfiles:\n%s", buf.String())),
	}
}

// CmdRemoteAdd adds a remote backend
func (m *managementConsole) CmdRemoteAdd(args unixsock.Args) *unixsock.Response {

	// Extract backend name
	required := []arg{
		arg{"backend", reflect.String},
	}

	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Connect to backend
	backend := args["backend"].(string)
	switch strings.ToLower(backend) {

	case "journald":

		required := []arg{
			arg{"host", reflect.String},
			arg{"port", reflect.Float64},
			arg{"service", reflect.String},
			arg{"instance", reflect.String},
			arg{"token", reflect.String},
		}

		if !validArguments(args, required) {
			return respMissingArgs
		}

		host := args["host"].(string)
		port := int(args["port"].(float64))
		service := args["service"].(string)
		instance := args["instance"].(string)
		token := args["token"].(string)

		backendKey := getCleanBackendKey("journald", host, port)

		m.logserver.Lock()
		defer m.logserver.Unlock()

		remote, err := journal.ConnectToJournald(host, port, service, instance, token, 10*time.Second)
		if err != nil {
			return &unixsock.Response{
				Status: unixsock.STATUS_FAIL,
				Error:  err.Error(),
			}
		}

		if err = m.logserver.logger.AddDestination(backendKey, remote); err != nil {
			return &unixsock.Response{
				Status: unixsock.STATUS_FAIL,
				Error:  err.Error(),
			}
		}

		return &unixsock.Response{
			Status:  unixsock.STATUS_OK,
			Payload: console(fmt.Sprintf("added remote backend %s", bold("journald"))),
		}

	case "kafka":
		return &unixsock.Response{
			Status: unixsock.STATUS_FAIL,
			Error:  fmt.Sprint("Not implemented yet"),
		}

	default:
		return &unixsock.Response{
			Status: unixsock.STATUS_FAIL,
			Error:  fmt.Sprintf("Unknown backend '%s'", backend),
		}
	}

}

// CmdRemoteRemove removes a remote backend
func (m *managementConsole) CmdRemoteRemove(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteList lists all active remote backends
func (m *managementConsole) CmdRemoteList(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}
