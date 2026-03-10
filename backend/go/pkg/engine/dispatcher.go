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
		return
	}
	defer conn.Close()

	fmt.Fprintln(conn, ":MEAS?")

	reader := bufio.NewReader(conn)
	response, _ := reader.ReadString('\n')

	e.Results <- Measurement{
		DeviceID: id,
		Data:     response,
		Latency:  time.Since(start),
	}
}
