package server

import (
	"fmt"
	"time"
	"reflect"
	"strings"
	"strconv"
	"sort"
	"github.com/fatih/color"

	rand "crypto/rand"
	"crypto/sha256"
)

// ManagementConsole handles commands received over the unix socket
type ManagementConsole interface {

	// CmdStatistics displays various statistics
	CmdStatistics(Args) *Response

	// CmdLogsList list all available logfiles and their archives
	CmdLogsList(Args) *Response

	// CmdRemoteAdd adds a remote backend
	CmdRemoteAdd(Args) *Response

	// CmdRemoteList lists all active remote backends
	CmdRemoteList(Args) *Response

	// CmdRemoteRemove removes a remote backend
	CmdRemoteRemove(Args) *Response

	// CmdTokensAdd adds a new token for a service/instance
	CmdTokensAdd(Args) *Response

	// CmdTokensListInstances lists all permitted instances of a service
	CmdTokensListInstances(Args) *Response

	// CmdTokensListServices lists all permitted services
	CmdTokensListServices(Args) *Response

	// CmdTokensRemoveInstance removes the token of a service/instance
	CmdTokensRemoveInstance(Args) *Response

	// CmdTokensRemoveService removes the token of all instances of a service
	CmdTokensRemoveService(Args) *Response

	// Execute is the executor of management console commands
	Execute(string, Args) *Response
}

// NewConsole creates a new management console for the log server
func NewConsole(server *LogServer) ManagementConsole {

	return &managementConsole{
		logserver: server,
	}
}

// managementConsole handles commands received over the unix socket
type managementConsole struct {
	banner string
	logserver *LogServer
}

// Execute is the executor of management console commands
func (m *managementConsole) Execute(cmd string, args Args) *Response {

	response := &Response{
		Status: "failure",
		Error:  fmt.Errorf("Execute: unknown command '%s'", cmd),
	}

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
		return response
	}

}

// arg is a helper struct used to for slices of required arguments
type arg struct {
	Name string
	Kind reflect.Kind
}

// validArguments verifies that all the required arguments have been provided
func validArguments(args Args, required []arg) bool {
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

var respMissingArgs = &Response{
	Status: "failure",
	Error:  fmt.Errorf("Missing/invalid parameters"),
}

// CmdStatistics displays various log-related statistics
func (m *managementConsole) CmdStatistics(args Args) * Response {
	return &Response{}
}

// CmdTokensAdd adds a new token for a service/instance
func (m *managementConsole) CmdTokensAdd(args Args) *Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
		arg{"instance", reflect.String},
	}

	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service/instance
	service := args["service"].(string)
	instance := args["instance"].(string)
	key := fmt.Sprintf("%s/%s", strings.ToLower(service), strings.ToLower(instance))

	// Create a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return &Response{
			Status: "failure",
			Error:  fmt.Errorf("Could not generate a new random token. Sorry"),
		}
	}
	token := fmt.Sprintf("%x", sha256.Sum256(tokenBytes))

	// TODO: implement locking of tokens
	// m.logserver.Lock()
	// defer m.logserver.Unlock()

	if _, ok := m.logserver.tokens[key]; !ok {
		m.logserver.tokens[key] = token
		// Write tokens to file
		// TODO
		return &Response{
			Status:  "success",
			Payload: fmt.Sprintf("Created a new token for %s/%s:\n%s", service, instance, boxStr(token)),
		}
	}

	return &Response{
		Status:  "failure",
		Payload: fmt.Sprintf("Token for %s/%s already exists", service, instance),
	}

}

// CmdTokensRemoveInstance removes the token of a service/instance
func (m *managementConsole) CmdTokensRemoveInstance(args Args) *Response {
	return &Response{}
}

// CmdTokensRemoveService removes the token of all instances of a service
func (m *managementConsole) CmdTokensRemoveService(args Args) *Response {
	return &Response{}
}

// CmdTokensListInstances lists all permitted instances of a service
func (m *managementConsole) CmdTokensListInstances(args Args) *Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
	}

	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Prepare table
	table := [][]string{}
	table = append(table,[]string{"Instance", "Token", "Last IP", "Logs parsed"})

	// Identify service
	service := strings.ToLower(args["service"].(string))

	for key, token := range m.logserver.tokens {
		parts := strings.Split(key,"/")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == service {
			table = append(table,[]string{parts[1], fmt.Sprintf("%s...",token[0:10]),"???","???"})
		}
	}

	return &Response{
		Status: "success",
		Payload: fmt.Sprintf("%s\nFollowing instances are permitted for service '%s':\n%s",m.logserver.GetBanner(), service, tableStr(table)),
	}
}

// CmdTokensListServices lists all permitted services
func (m *managementConsole) CmdTokensListServices(args Args) *Response {

	// Prepare statistics
	serviceNames := []string{}
	services := map[string][2]int{}
	for key, _ := range m.logserver.tokens {
		parts := strings.Split(key,"/")
		if len(parts) != 2 {
			continue
		}
		if _,ok := services[parts[0]]; !ok {
			serviceNames = append(serviceNames,parts[0])
			services[parts[0]] = [2]int{}
		}
		counts := services[parts[0]]
		counts[0]++
	}
	sort.Strings(serviceNames)


	red := color.New(color.FgRed)

	// Prepare table
	table := [][]string{}
	table = append(table,[]string{red.Sprint("●"),"Service", "Instances", "Last log entry", "Log entries parsed"})
	for _, name := range serviceNames {
		service := services[name]
		entry := []string{
			red.Sprint("●"),
			name,
			strconv.Itoa(service[0]),
			time.Now().Format("2006-01-02 15:04"),
			strconv.Itoa(service[1])+"/0Mb",
		}
		table = append(table,entry)
	}

	return &Response{
		Status: "success",
		Payload: fmt.Sprintf("%s\nFollowing services are permitted:\n%s",m.logserver.GetBanner(), tableStr(table)),
	}
}

// CmdLogsList list all available logfiles and their archives
func (m *managementConsole) CmdLogsList(args Args) *Response {
	return &Response{}
}

// CmdRemoteAdd adds a remote backend
func (m *managementConsole) CmdRemoteAdd(args Args) *Response {
	return &Response{}
}

// CmdRemoteRemove removes a remote backend
func (m *managementConsole) CmdRemoteRemove(args Args) *Response {
	return &Response{}
}

// CmdRemoteList lists all active remote backends
func (m *managementConsole) CmdRemoteList(args Args) *Response {
	return &Response{}
}
