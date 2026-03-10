package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type VirtualEquipment struct {
	ID int
}

func (ve *VirtualEquipment) PerformCalculation() (float64, float64) {
	// Simulate a live calculation (e.g., measuring voltage and current with noise)
	baseVoltage := 3.3
	noise := (rand.Float64() - 0.5) * 0.05
	voltage := baseVoltage + noise

	resistance := 100.0 + (rand.Float64()-0.5)*1.0
	current := voltage / resistance

	// Simulate processing delay
	time.Sleep(time.Duration(rand.Intn(5)+1) * time.Millisecond)

	return voltage, current
}

type BenchmarkResponse struct {
	Device    string `json:"device"`
	GoLatency string `json:"goLatency"`
	GoData    string `json:"goData"`
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func benchmarkHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		return
	}

	countStr := r.URL.Query().Get("count")
	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 {
		count = 1
	}

	results := make([]BenchmarkResponse, 0, count)

	for i := 0; i < count; i++ {
		equip := VirtualEquipment{ID: i + 1}

		start := time.Now()
		v, c := equip.PerformCalculation()
		duration := time.Since(start)

		results = append(results, BenchmarkResponse{
			Device:    fmt.Sprintf("SMU-%d", i+1),
			GoLatency: fmt.Sprintf("%.3f", float64(duration.Microseconds())/1000.0),
			GoData:    fmt.Sprintf("V:%.4f,I:%.4f", v, c),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/benchmark", benchmarkHandler)
	fmt.Println("Go Virtual Equipment Server running on port 8080")
	http.ListenAndServe(":8080", nil)
}
