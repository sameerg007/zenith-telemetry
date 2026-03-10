package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"zenith-telemetry/pkg/engine"
	"zenith-telemetry/pkg/simulator"
)

const (
	maxInstruments     = 50
	instrumentPortBase = 9001
	serverPort         = "8080"
)

type BenchmarkResponse struct {
	Device    string `json:"device"`
	GoLatency string `json:"goLatency"`
	GoData    string `json:"goData"`
}

type BenchmarkResult struct {
	CycleTimeMs  string              `json:"cycleTimeMs"`
	Measurements []BenchmarkResponse `json:"measurements"`
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func benchmarkHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || count < 1 {
		count = 5
	}
	if count > maxInstruments {
		count = maxInstruments
	}

	zEngine := &engine.ZenithEngine{
		Results: make(chan engine.Measurement, count),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	cycleStart := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		id := fmt.Sprintf("SMU-%d", i+1)
		addr := fmt.Sprintf("localhost:%d", instrumentPortBase+i)
		go zEngine.Poll(ctx, id, addr, &wg)
	}

	go func() {
		wg.Wait()
		close(zEngine.Results)
	}()

	var measurements []BenchmarkResponse
	for res := range zEngine.Results {
		measurements = append(measurements, BenchmarkResponse{
			Device:    res.DeviceID,
			GoLatency: fmt.Sprintf("%.3f", float64(res.Latency.Microseconds())/1000.0),
			GoData:    strings.TrimSpace(res.Data),
		})
	}

	cycleMs := float64(time.Since(cycleStart).Microseconds()) / 1000.0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(BenchmarkResult{
		CycleTimeMs:  fmt.Sprintf("%.3f", cycleMs),
		Measurements: measurements,
	})
}

func main() {
	for i := 0; i < maxInstruments; i++ {
		port := strconv.Itoa(instrumentPortBase + i)
		go simulator.StartMockInstrument(port)
	}
	time.Sleep(100 * time.Millisecond)

	fmt.Println("--- ZENITH GO PARALLEL ENGINE ---")
	fmt.Printf("Mock Instruments : %d active (ports %d-%d)\n",
		maxInstruments, instrumentPortBase, instrumentPortBase+maxInstruments-1)
	fmt.Printf("HTTP Server      : http://localhost:%s/benchmark?count=N\n", serverPort)

	http.HandleFunc("/benchmark", benchmarkHandler)
	if err := http.ListenAndServe(":"+serverPort, nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
