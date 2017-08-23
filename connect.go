package log

import (
  "io"
  "time"
)

// ConnectToLogServer connects to a log server backend
func ConnectToLogServer(host string, port int, token string, timeout time.Duration) (io.Writer, error) {

  return nil, nil
}

// ConnectToKafka connects to a kafka backend as a producer
func ConnectToKafka(host string, port int, topic string) (io.Writer, error) {

  return nil, nil
}
