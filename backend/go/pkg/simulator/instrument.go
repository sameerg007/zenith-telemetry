package simulator

import (
	"bufio"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"strings"
)

func StartMockInstrument(port string) {
	// Generate stable readings once for this virtual instrument at startup.
	// Every subsequent :MEAS? poll on this port returns the same V/I so that
	// both the Go engine and the Python engine (which share the same ports)
	// report identical measurement data — isolating latency as the only variable.
	v := 0.8 + rand.Float64()*(2.1-0.2)
	i := 0.01 + rand.Float64()*(0.59-0.01)
	measResponse := fmt.Sprintf("V:%.4f,I:%.4f", v, i)

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		slog.Error("mock instrument failed to start", "port", port, "error", err)
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("mock instrument accept error", "port", port, "error", err)
			return
		}
		go handleConnection(conn, measResponse)
	}
}

func handleConnection(conn net.Conn, measResponse string) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		switch strings.TrimSpace(scanner.Text()) {
		case "*IDN?":
			fmt.Fprintln(conn, "ZENITH-MOCK-B2901A-V2.6")
		case ":MEAS?":
			fmt.Fprintln(conn, measResponse)
		default:
			fmt.Fprintln(conn, "ERR:INVALID_SCPI_CMD")
		}
	}
}
