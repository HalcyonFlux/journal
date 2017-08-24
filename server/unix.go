package server

import (
  "net"
)

// HandleUnixRequest handles a request via a unix socket connection
func (l *LogServer) HandleUnixRequest(conn net.Conn) {
	defer conn.Close()

}
