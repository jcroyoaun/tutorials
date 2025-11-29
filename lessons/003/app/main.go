// Package main implements a chaos engineering resource consumer for Kubernetes.
// It allows dynamic control of CPU, memory, latency, and health probe behavior
// via HTTP API, with Prometheus metrics and expvar support.
package main

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// =============================================================================
// Configuration

type config struct {
	Host            string
	Port            string
	ShutdownTimeout time.Duration
}

func newConfig() config {
	return config{
		Host:            getEnv("HOST", "0.0.0.0"),
		Port:            getEnv("PORT", "8080"),
		ShutdownTimeout: 10 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// =============================================================================
// Metrics (expvar style, a la Bill Kennedy)

var metrics = struct {
	goroutines    *expvar.Int
	requests      *expvar.Int
	memoryTarget  *expvar.Int
	memoryActual  *expvar.Int
	cpuTarget     *expvar.Int
	cpuActual     *expvar.Float
	latencyTarget *expvar.Int
	errorRate     *expvar.Float
}{
	goroutines:    expvar.NewInt("goroutines"),
	requests:      expvar.NewInt("requests"),
	memoryTarget:  expvar.NewInt("memory_target_mb"),
	memoryActual:  expvar.NewInt("memory_actual_mb"),
	cpuTarget:     expvar.NewInt("cpu_target_millicores"),
	cpuActual:     expvar.NewFloat("cpu_actual_percent"),
	latencyTarget: expvar.NewInt("latency_target_ms"),
	errorRate:     expvar.NewFloat("error_rate"),
}

// =============================================================================
// Resource Controller

type pattern string

const (
	patternConstant pattern = "constant"
	patternSine     pattern = "sine"
	patternSpike    pattern = "spike"
	patternRamp     pattern = "ramp"
)

type resourceController struct {
	mu sync.RWMutex

	// Memory
	memoryTargetMB int
	memoryRampSec  int
	memoryHolder   []byte
	memoryStarted  time.Time

	// CPU
	cpuTargetMillicores int
	cpuPattern          pattern
	cpuCancel           context.CancelFunc

	// Latency injection
	latencyMS       int
	latencyJitterMS int

	// Error injection
	errorRate float64

	// Health probes
	ready atomic.Bool
	live  atomic.Bool

	log *slog.Logger
}

func newResourceController(log *slog.Logger) *resourceController {
	rc := &resourceController{
		cpuPattern: patternConstant,
		log:        log,
	}
	rc.ready.Store(true)
	rc.live.Store(true)
	return rc
}

// -----------------------------------------------------------------------------
// Memory Management

type memoryRequest struct {
	MB          int `json:"mb"`
	RampSeconds int `json:"ramp_seconds"`
}

func (rc *resourceController) setMemory(mb, rampSeconds int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.memoryTargetMB = mb
	rc.memoryRampSec = rampSeconds
	rc.memoryStarted = time.Now()

	metrics.memoryTarget.Set(int64(mb))

	if rampSeconds <= 0 {
		rc.allocateMemory(mb)
	} else {
		go rc.rampMemory()
	}

	rc.log.Info("memory target set", "mb", mb, "ramp_seconds", rampSeconds)
}

func (rc *resourceController) rampMemory() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		elapsed := time.Since(rc.memoryStarted).Seconds()
		progress := elapsed / float64(rc.memoryRampSec)

		if progress >= 1.0 {
			rc.allocateMemory(rc.memoryTargetMB)
			rc.mu.Unlock()
			return
		}

		currentMB := int(float64(rc.memoryTargetMB) * progress)
		rc.allocateMemory(currentMB)
		rc.mu.Unlock()
	}
}

func (rc *resourceController) allocateMemory(mb int) {
	size := mb * 1024 * 1024
	if size <= 0 {
		rc.memoryHolder = nil
		metrics.memoryActual.Set(0)
		return
	}

	rc.memoryHolder = make([]byte, size)
	// Touch every page to force actual allocation
	for i := 0; i < size; i += 4096 {
		rc.memoryHolder[i] = 1
	}
	metrics.memoryActual.Set(int64(mb))
}

// -----------------------------------------------------------------------------
// CPU Management

type cpuRequest struct {
	Millicores int    `json:"millicores"`
	Pattern    string `json:"pattern"`
}

func (rc *resourceController) setCPU(millicores int, p pattern) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Cancel existing CPU consumers
	if rc.cpuCancel != nil {
		rc.cpuCancel()
	}

	rc.cpuTargetMillicores = millicores
	rc.cpuPattern = p
	metrics.cpuTarget.Set(int64(millicores))

	if millicores <= 0 {
		rc.log.Info("cpu consumption stopped")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	rc.cpuCancel = cancel

	go rc.consumeCPU(ctx)
	rc.log.Info("cpu target set", "millicores", millicores, "pattern", p)
}

func (rc *resourceController) consumeCPU(ctx context.Context) {
	rc.mu.RLock()
	millicores := rc.cpuTargetMillicores
	p := rc.cpuPattern
	rc.mu.RUnlock()

	numCores := millicores / 1000
	remainder := millicores % 1000

	// Spawn goroutines for full cores
	for i := 0; i < numCores; i++ {
		go func() {
			runtime.LockOSThread()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Burn CPU
				}
			}
		}()
	}

	// Fractional core with pattern
	if remainder > 0 {
		go func() {
			runtime.LockOSThread()
			period := 100 * time.Millisecond
			startTime := time.Now()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					var fraction float64
					switch p {
					case patternSine:
						// Sine wave: oscillates between 0 and target
						elapsed := time.Since(startTime).Seconds()
						// Full cycle every 60 seconds
						fraction = (float64(remainder) / 1000.0) * (0.5 + 0.5*math.Sin(elapsed*2*math.Pi/60))
					case patternSpike:
						// Random spikes
						elapsed := time.Since(startTime).Seconds()
						if int(elapsed)%10 < 2 { // Spike for 2 seconds every 10
							fraction = float64(remainder) / 1000.0
						} else {
							fraction = float64(remainder) / 1000.0 * 0.1
						}
					default: // constant
						fraction = float64(remainder) / 1000.0
					}

					metrics.cpuActual.Set(fraction * 100)

					busyTime := time.Duration(float64(period) * fraction)
					sleepTime := period - busyTime

					start := time.Now()
					for time.Since(start) < busyTime {
						// Busy spin
					}
					time.Sleep(sleepTime)
				}
			}
		}()
	}
}

// -----------------------------------------------------------------------------
// Latency Injection

type latencyRequest struct {
	MS       int `json:"ms"`
	JitterMS int `json:"jitter_ms"`
}

func (rc *resourceController) setLatency(ms, jitterMS int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.latencyMS = ms
	rc.latencyJitterMS = jitterMS
	metrics.latencyTarget.Set(int64(ms))

	rc.log.Info("latency set", "ms", ms, "jitter_ms", jitterMS)
}

func (rc *resourceController) injectLatency() {
	rc.mu.RLock()
	ms := rc.latencyMS
	jitter := rc.latencyJitterMS
	rc.mu.RUnlock()

	if ms <= 0 {
		return
	}

	delay := time.Duration(ms) * time.Millisecond
	if jitter > 0 {
		// Simple jitter: add random amount up to jitter
		jitterNs := time.Now().UnixNano() % int64(jitter*int(time.Millisecond))
		delay += time.Duration(jitterNs)
	}
	time.Sleep(delay)
}

// -----------------------------------------------------------------------------
// Error Injection

type errorRequest struct {
	Rate float64 `json:"rate"`
}

func (rc *resourceController) setErrorRate(rate float64) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.errorRate = rate
	metrics.errorRate.Set(rate)

	rc.log.Info("error rate set", "rate", rate)
}

func (rc *resourceController) shouldError() bool {
	rc.mu.RLock()
	rate := rc.errorRate
	rc.mu.RUnlock()

	if rate <= 0 {
		return false
	}

	// Use nanoseconds as cheap random
	return float64(time.Now().UnixNano()%100)/100.0 < rate
}

// -----------------------------------------------------------------------------
// Health Probe Control

type healthRequest struct {
	Ready *bool `json:"ready,omitempty"`
	Live  *bool `json:"live,omitempty"`
}

func (rc *resourceController) setHealth(ready, live *bool) {
	if ready != nil {
		rc.ready.Store(*ready)
		rc.log.Info("readiness set", "ready", *ready)
	}
	if live != nil {
		rc.live.Store(*live)
		rc.log.Info("liveness set", "live", *live)
	}
}

// -----------------------------------------------------------------------------
// Status

type status struct {
	Memory struct {
		TargetMB  int     `json:"target_mb"`
		ActualMB  int     `json:"actual_mb"`
		RampSec   int     `json:"ramp_seconds"`
		Progress  float64 `json:"progress"`
	} `json:"memory"`
	CPU struct {
		TargetMillicores int     `json:"target_millicores"`
		Pattern          string  `json:"pattern"`
		ActualPercent    float64 `json:"actual_percent"`
	} `json:"cpu"`
	Latency struct {
		MS       int `json:"ms"`
		JitterMS int `json:"jitter_ms"`
	} `json:"latency"`
	ErrorRate float64 `json:"error_rate"`
	Health    struct {
		Ready bool `json:"ready"`
		Live  bool `json:"live"`
	} `json:"health"`
	Goroutines int `json:"goroutines"`
}

func (rc *resourceController) getStatus() status {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var s status

	s.Memory.TargetMB = rc.memoryTargetMB
	s.Memory.ActualMB = len(rc.memoryHolder) / (1024 * 1024)
	s.Memory.RampSec = rc.memoryRampSec
	if rc.memoryRampSec > 0 {
		s.Memory.Progress = math.Min(1.0, time.Since(rc.memoryStarted).Seconds()/float64(rc.memoryRampSec))
	} else {
		s.Memory.Progress = 1.0
	}

	s.CPU.TargetMillicores = rc.cpuTargetMillicores
	s.CPU.Pattern = string(rc.cpuPattern)
	s.CPU.ActualPercent = metrics.cpuActual.Value()

	s.Latency.MS = rc.latencyMS
	s.Latency.JitterMS = rc.latencyJitterMS

	s.ErrorRate = rc.errorRate

	s.Health.Ready = rc.ready.Load()
	s.Health.Live = rc.live.Load()

	s.Goroutines = runtime.NumGoroutine()

	return s
}

// =============================================================================
// HTTP Handlers

type handlers struct {
	rc  *resourceController
	log *slog.Logger
}

func (h *handlers) handleMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req memoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.rc.setMemory(req.MB, req.RampSeconds)
	h.respond(w, map[string]any{"status": "ok", "memory_mb": req.MB, "ramp_seconds": req.RampSeconds})
}

func (h *handlers) handleCPU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req cpuRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := patternConstant
	if req.Pattern != "" {
		p = pattern(req.Pattern)
	}

	h.rc.setCPU(req.Millicores, p)
	h.respond(w, map[string]any{"status": "ok", "millicores": req.Millicores, "pattern": p})
}

func (h *handlers) handleLatency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req latencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.rc.setLatency(req.MS, req.JitterMS)
	h.respond(w, map[string]any{"status": "ok", "latency_ms": req.MS, "jitter_ms": req.JitterMS})
}

func (h *handlers) handleErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req errorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.rc.setErrorRate(req.Rate)
	h.respond(w, map[string]any{"status": "ok", "error_rate": req.Rate})
}

func (h *handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req healthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.rc.setHealth(req.Ready, req.Live)
	h.respond(w, map[string]any{"status": "ok"})
}

func (h *handlers) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics.goroutines.Set(int64(runtime.NumGoroutine()))
	h.respond(w, h.rc.getStatus())
}

func (h *handlers) handleReady(w http.ResponseWriter, r *http.Request) {
	h.rc.injectLatency()

	if h.rc.shouldError() {
		http.Error(w, "injected error", http.StatusInternalServerError)
		return
	}

	if !h.rc.ready.Load() {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func (h *handlers) handleLive(w http.ResponseWriter, r *http.Request) {
	if !h.rc.live.Load() {
		http.Error(w, "not live", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("live"))
}

func (h *handlers) handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics.goroutines.Set(int64(runtime.NumGoroutine()))

	// Prometheus format
	fmt.Fprintf(w, "# HELP resource_hog_goroutines Number of goroutines\n")
	fmt.Fprintf(w, "# TYPE resource_hog_goroutines gauge\n")
	fmt.Fprintf(w, "resource_hog_goroutines %d\n", metrics.goroutines.Value())

	fmt.Fprintf(w, "# HELP resource_hog_memory_target_mb Target memory in MB\n")
	fmt.Fprintf(w, "# TYPE resource_hog_memory_target_mb gauge\n")
	fmt.Fprintf(w, "resource_hog_memory_target_mb %d\n", metrics.memoryTarget.Value())

	fmt.Fprintf(w, "# HELP resource_hog_memory_actual_mb Actual memory allocated in MB\n")
	fmt.Fprintf(w, "# TYPE resource_hog_memory_actual_mb gauge\n")
	fmt.Fprintf(w, "resource_hog_memory_actual_mb %d\n", metrics.memoryActual.Value())

	fmt.Fprintf(w, "# HELP resource_hog_cpu_target_millicores Target CPU in millicores\n")
	fmt.Fprintf(w, "# TYPE resource_hog_cpu_target_millicores gauge\n")
	fmt.Fprintf(w, "resource_hog_cpu_target_millicores %d\n", metrics.cpuTarget.Value())

	fmt.Fprintf(w, "# HELP resource_hog_cpu_actual_percent Actual CPU usage percent\n")
	fmt.Fprintf(w, "# TYPE resource_hog_cpu_actual_percent gauge\n")
	fmt.Fprintf(w, "resource_hog_cpu_actual_percent %.2f\n", metrics.cpuActual.Value())

	fmt.Fprintf(w, "# HELP resource_hog_latency_target_ms Target latency injection in ms\n")
	fmt.Fprintf(w, "# TYPE resource_hog_latency_target_ms gauge\n")
	fmt.Fprintf(w, "resource_hog_latency_target_ms %d\n", metrics.latencyTarget.Value())

	fmt.Fprintf(w, "# HELP resource_hog_error_rate Injected error rate\n")
	fmt.Fprintf(w, "# TYPE resource_hog_error_rate gauge\n")
	fmt.Fprintf(w, "resource_hog_error_rate %.2f\n", metrics.errorRate.Value())

	fmt.Fprintf(w, "# HELP resource_hog_requests_total Total requests\n")
	fmt.Fprintf(w, "# TYPE resource_hog_requests_total counter\n")
	fmt.Fprintf(w, "resource_hog_requests_total %d\n", metrics.requests.Value())
}

func (h *handlers) respond(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// =============================================================================
// Middleware

func requestCounter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.requests.Add(1)
		next.ServeHTTP(w, r)
	})
}

func logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start),
			)
		})
	}
}

// =============================================================================
// Main

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := newConfig()

	rc := newResourceController(log)
	h := &handlers{rc: rc, log: log}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/memory", h.handleMemory)
	mux.HandleFunc("/api/cpu", h.handleCPU)
	mux.HandleFunc("/api/latency", h.handleLatency)
	mux.HandleFunc("/api/errors", h.handleErrors)
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/status", h.handleStatus)

	// Health probes
	mux.HandleFunc("/healthz", h.handleLive)
	mux.HandleFunc("/readyz", h.handleReady)

	// Observability
	mux.HandleFunc("/metrics", h.handleMetrics)
	mux.Handle("/debug/vars", expvar.Handler())

	// Apply middleware
	handler := requestCounter(logging(log)(mux))

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler: handler,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("shutting down server")
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		server.Shutdown(ctx)
	}()

	log.Info("starting server", "addr", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
}
