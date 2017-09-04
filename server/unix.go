package server

import (
	"net"
)

// HandleUnixRequest handles a request via a unix socket connection
// It expects to read only a single message and respond to it immediately
func (l *LogServer) HandleUnixRequest(c net.Conn) {
	defer c.Close()

	manager := NewConsole(l)

Loop:
	for {

		// Receive the command
		receiver := NewReceiver(c)
		if err := receiver.Receive(); err != nil {
			break Loop
		}

		// Handle the command
		response := manager.Execute(receiver.GetCmd(), receiver.GetArgs())

		// Respond
		if receiver.ShouldRespond() {
			receiver.SetResponse(response)
			receiver.Send()
		}

		// Close connection
		if receiver.ShouldClose() {
			break Loop
		}

	}

}
