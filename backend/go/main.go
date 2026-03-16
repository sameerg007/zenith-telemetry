package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	"zenith-telemetry/pkg/engine"
	"zenith-telemetry/pkg/simulator"
)

const (
	maxInstruments     = 50
	instrumentPortBase = 9001
	defaultServerPort  = "8080"
	requestTimeout     = 10 * time.Second
	startupDelay       = 150 * time.Millisecond
	maxConcurrentPolls = 100 // Worker pool size
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

// allowedOrigin returns the CORS origin from the environment, defaulting to localhost.
func allowedOrigin() string {
	if o := os.Getenv("ALLOWED_ORIGIN"); o != "" {
		return o
	}
	return "http://localhost:3000"
}

// corsMiddleware restricts cross-origin access to a configurable origin.
func corsMiddleware(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

func benchmarkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// If 'count' is omitted, use the default. If it is present but invalid,
	// return 400 rather than silently defaulting — explicit errors are safer
	// and easier to debug than silent coercions.
	var count int
	if raw := r.URL.Query().Get("count"); raw == "" {
		count = 5
	} else {
		var err error
		count, err = strconv.Atoi(raw)
		if err != nil || count < 1 || count > maxInstruments {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w,
				fmt.Sprintf(`{"error":"'count' must be an integer between 1 and %d"}`, maxInstruments),
				http.StatusBadRequest,
			)
			return
		}
	}

	zEngine := &engine.ZenithEngine{}

	// Honour both the global request timeout and any client-side cancellation.
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	var wg sync.WaitGroup
	cycleStart := time.Now()

	// Worker pool: buffered channel to limit concurrent goroutines.
	semaphore := make(chan struct{}, maxConcurrentPolls)
	results := make([]BenchmarkResponse, count)

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			// Acquire a worker slot inside the goroutine so the loop never
			// blocks the main goroutine — all goroutines are spawned immediately
			// and queue themselves for execution.
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			id := fmt.Sprintf("SMU-%d", i+1)
			addr := fmt.Sprintf("localhost:%d", instrumentPortBase+i)

			res, err := zEngine.Poll(ctx, id, addr)
			if err != nil {
				slog.Warn("poll error", "device", id, "error", err)
			}

			// No mutex needed: each goroutine writes to its own unique index.
			results[i] = BenchmarkResponse{
				Device:    res.DeviceID,
				GoLatency: fmt.Sprintf("%.3f", float64(res.Latency.Microseconds())/1000.0),
				GoData:    res.Data, // Already trimmed and validated in dispatcher.
			}
		}(i)
	}

	wg.Wait()
	cycleMs := float64(time.Since(cycleStart).Microseconds()) / 1000.0

	w.Header().Set("Content-Type", "application/json")
	// Prevent intermediary caches from serving stale benchmark data.
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(BenchmarkResult{
		CycleTimeMs:  fmt.Sprintf("%.3f", cycleMs),
		Measurements: results,
	}); err != nil {
		slog.Error("failed to encode benchmark response", "error", err)
	}
}

func serverPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultServerPort
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	for i := 0; i < maxInstruments; i++ {
		go simulator.StartMockInstrument(strconv.Itoa(instrumentPortBase + i))
	}
	time.Sleep(startupDelay)

	origin := allowedOrigin()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/benchmark", benchmarkHandler)

	port := serverPort()
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(origin, mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("ZENITH GO PARALLEL ENGINE",
		"instruments", maxInstruments,
		"port_range", fmt.Sprintf("%d-%d", instrumentPortBase, instrumentPortBase+maxInstruments-1),
		"listen", fmt.Sprintf("http://localhost:%s", port),
		"allowed_origin", origin,
	)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server…")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
