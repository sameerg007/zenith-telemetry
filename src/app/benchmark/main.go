// package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"math/rand"
// 	"net/http"
// 	"strconv"
// 	"time"
// )

// type VirtualEquipment struct {
// 	ID int
// }

// func (ve *VirtualEquipment) PerformCalculation() (float64, float64) {
// 	// Simulate a live calculation (e.g., measuring voltage and current with noise)
// 	baseVoltage := 3.3
// 	noise := (rand.Float64() - 0.5) * 0.05
// 	voltage := baseVoltage + noise

// 	resistance := 100.0 + (rand.Float64()-0.5)*1.0
// 	current := voltage / resistance

// 	// Simulate processing delay
// 	time.Sleep(time.Duration(rand.Intn(5)+1) * time.Millisecond)

// 	return voltage, current
// }

// type BenchmarkResponse struct {
// 	Device    string `json:"device"`
// 	GoLatency string `json:"goLatency"`
// 	GoData    string `json:"goData"`
// }

// func enableCors(w http.ResponseWriter) {
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
// 	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
// }

// func benchmarkHandler(w http.ResponseWriter, r *http.Request) {
// 	enableCors(w)
// 	if r.Method == "OPTIONS" {
// 		return
// 	}

// 	countStr := r.URL.Query().Get("count")
// 	count, err := strconv.Atoi(countStr)
// 	if err != nil || count < 1 {
// 		count = 1
// 	}

// 	results := make([]BenchmarkResponse, 0, count)

// 	for i := 0; i < count; i++ {
// 		equip := VirtualEquipment{ID: i + 1}

// 		start := time.Now()
// 		v, c := equip.PerformCalculation()
// 		duration := time.Since(start)

// 		results = append(results, BenchmarkResponse{
// 			Device:    fmt.Sprintf("SMU-%d", i+1),
// 			GoLatency: fmt.Sprintf("%.3f", float64(duration.Microseconds())/1000.0),
// 			GoData:    fmt.Sprintf("V:%.4f,I:%.4f", v, c),
// 		})
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(results)
// }

//	func main() {
//		rand.Seed(time.Now().UnixNano())
//		http.HandleFunc("/benchmark", benchmarkHandler)
//		fmt.Println("Go Virtual Equipment Server running on port 8080")
//		http.ListenAndServe(":8080", nil)
//	}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type VirtualEquipment struct {
	ID int
}

// 1. CONTEXT INTEGRATION: Simulates hardware poll but respects timeouts
func (ve *VirtualEquipment) PerformCalculation(ctx context.Context) (float64, float64, time.Duration, error) {
	start := time.Now()

	// Simulate hardware I/O latency
	delay := time.Duration(rand.Intn(5)+1) * time.Millisecond

	// Use select to listen for either the hardware finishing, or the context timing out
	select {
	case <-time.After(delay):
		baseVoltage := 3.3
		voltage := baseVoltage + (rand.Float64()-0.5)*0.05
		current := voltage / (100.0 + (rand.Float64()-0.5)*1.0)
		return voltage, current, time.Since(start), nil
	case <-ctx.Done():
		// If the external device hangs, we abort cleanly rather than leaking the Goroutine
		return 0, 0, 0, ctx.Err()
	}
}

type BenchmarkResponse struct {
	Device    string `json:"device"`
	GoLatency string `json:"goLatency"`
	GoData    string `json:"goData"`
	Error     string `json:"error,omitempty"` // Added error field for timeouts
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

	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || count < 1 {
		count = 1
	}

	// 2. TIMEOUT PROTECTION: Hard stop at 2 seconds so the server never hangs
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Pre-allocate the slice
	results := make([]BenchmarkResponse, count)
	var wg sync.WaitGroup

	// 3. THE SEMAPHORE (WORKER POOL): Prevents DDOS by limiting concurrent workers to 100
	maxWorkers := 100
	semaphore := make(chan struct{}, maxWorkers)

	engineStart := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Acquire a token before starting work. If 100 are busy, it waits here.
			semaphore <- struct{}{}

			// Ensure we release the token when the function exits
			defer func() { <-semaphore }()

			equip := VirtualEquipment{ID: id}
			v, c, latency, err := equip.PerformCalculation(ctx)

			// 4. LOCK-FREE WRITE: No Mutex needed because we write to a unique, pre-allocated index
			if err != nil {
				results[id-1] = BenchmarkResponse{
					Device: fmt.Sprintf("SMU-%d", id),
					Error:  "Timeout/Error",
				}
				return
			}

			results[id-1] = BenchmarkResponse{
				Device:    fmt.Sprintf("SMU-%d", id),
				GoLatency: fmt.Sprintf("%.3f", float64(latency.Microseconds())/1000.0),
				GoData:    fmt.Sprintf("V:%.4f,I:%.4f", v, c),
			}
		}(i + 1)
	}

	wg.Wait()

	fmt.Printf("Engine Batch Processed %d devices in %v\n", count, time.Since(engineStart))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/benchmark", benchmarkHandler)

	// 5. SECURE HTTP SERVER: Mitigates Slowloris attacks with strict timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	fmt.Println("Production Zenith Telemetry Server running on port 8080")
	if err := srv.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
