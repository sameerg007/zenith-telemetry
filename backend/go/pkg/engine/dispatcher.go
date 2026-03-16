package engine

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	// dialTimeout caps how long we wait for the TCP handshake to a mock instrument.
	dialTimeout = 2 * time.Second
	// roundTripDeadline caps the full send+receive cycle per poll.
	roundTripDeadline = 3 * time.Second
	// maxResponseBytes prevents a misbehaving instrument from flooding the reader.
	maxResponseBytes = 512
)

type Measurement struct {
	DeviceID string
	Data     string
	Latency  time.Duration
	Err      error
}

// ZenithEngine dispatches concurrent polls to mock TCP instruments.
type ZenithEngine struct{}

func (e *ZenithEngine) Poll(ctx context.Context, id string, addr string) (Measurement, error) {
	start := time.Now()

	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)},
			fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Bound the full round-trip so a slow instrument cannot stall a goroutine.
	if err := conn.SetDeadline(time.Now().Add(roundTripDeadline)); err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)},
			fmt.Errorf("set deadline: %w", err)
	}

	if _, err := fmt.Fprintln(conn, ":MEAS?"); err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)},
			fmt.Errorf("write: %w", err)
	}

	// LimitedReader caps bytes read so a rogue instrument cannot cause unbounded allocation.
	resp, err := bufio.NewReader(&io.LimitedReader{R: conn, N: maxResponseBytes}).ReadString('\n')
	if err != nil {
		return Measurement{DeviceID: id, Data: "ERROR", Latency: time.Since(start)},
			fmt.Errorf("read: %w", err)
	}

	return Measurement{DeviceID: id, Data: resp, Latency: time.Since(start)}, nil
}
