package engine

import (
	"bufio"
	"context"
	"fmt"
	"net"
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

func (e *ZenithEngine) Poll(ctx context.Context, id string, addr string) (Measurement, error) {
	start := time.Now()

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Measurement{
			DeviceID: id,
			Data:     "ERROR",
			Latency:  time.Since(start),
		}, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Bound the total round-trip to 3s so a slow instrument can't hold up a goroutine.
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)}, fmt.Errorf("set deadline: %w", err)
	}

	if _, err := fmt.Fprintln(conn, ":MEAS?"); err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)}, fmt.Errorf("write: %w", err)
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)}, fmt.Errorf("read: %w", err)
	}

	return Measurement{
		DeviceID: id,
		Data:     response,
		Latency:  time.Since(start),
	}, nil
}
