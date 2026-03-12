package engine

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type Measurement struct {
	DeviceID string
	Data     string
	Latency  time.Duration
	Err      error
}

type ZenithEngine struct {
	Results chan Measurement
}

func (e *ZenithEngine) Poll(ctx context.Context, id string, addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	start := time.Now()

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		e.Results <- Measurement{
			DeviceID: id,
			Data:     "ERROR",
			Latency:  time.Since(start),
			Err:      fmt.Errorf("dial: %w", err),
		}
		return
	}
	defer conn.Close()

	// Bound the total round-trip to 3 s so a slow instrument can't hold up a goroutine.
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		e.Results <- Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start),
			Err: fmt.Errorf("set deadline: %w", err)}
		return
	}

	if _, err := fmt.Fprintln(conn, ":MEAS?"); err != nil {
		e.Results <- Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start),
			Err: fmt.Errorf("write: %w", err)}
		return
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		e.Results <- Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start),
			Err: fmt.Errorf("read: %w", err)}
		return
	}

	e.Results <- Measurement{
		DeviceID: id,
		Data:     response,
		Latency:  time.Since(start),
	}
}
