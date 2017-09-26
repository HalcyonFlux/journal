package connect

import "net"

// Returns the IP
// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func getIP() string {

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "N/A"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()

}
