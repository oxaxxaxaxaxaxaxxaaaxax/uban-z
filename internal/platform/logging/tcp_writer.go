package logging

import (
	"net"
	"sync"
	"time"
)

const (
	logstashQueueSize      = 1024
	logstashConnectTimeout = 2 * time.Second
	logstashRetryDelay     = 1 * time.Second
)

type asyncTCPWriter struct {
	addr string
	ch   chan []byte
	once sync.Once
}

func newAsyncTCPWriter(addr string) *asyncTCPWriter {
	w := &asyncTCPWriter{
		addr: addr,
		ch:   make(chan []byte, logstashQueueSize),
	}
	w.once.Do(func() {
		go w.run()
	})
	return w
}

func (w *asyncTCPWriter) Write(p []byte) (int, error) {
	msg := append([]byte(nil), p...)
	select {
	case w.ch <- msg:
	default:
	}
	return len(p), nil
}

func (w *asyncTCPWriter) run() {
	var conn net.Conn
	for msg := range w.ch {
		for {
			if conn == nil {
				next, err := net.DialTimeout("tcp", w.addr, logstashConnectTimeout)
				if err != nil {
					time.Sleep(logstashRetryDelay)
					continue
				}
				conn = next
			}

			if _, err := conn.Write(msg); err != nil {
				_ = conn.Close()
				conn = nil
				time.Sleep(logstashRetryDelay)
				continue
			}
			break
		}
	}
}
