package simulator

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
)

func StartMockInstrument(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		switch command {
		case "*IDN?":
			fmt.Fprintln(conn, "ZENITH-MOCK-B2901A-V2.6")
		case ":MEAS?":
			v := 0.8 + rand.Float64()*(1.2-0.8)
			i := 0.01 + rand.Float64()*(0.05-0.01)
			fmt.Fprintf(conn, "V:%.4f,I:%.4f\n", v, i)
		default:
			fmt.Fprintln(conn, "ERR:INVALID_SCPI_CMD")
		}
	}
}
