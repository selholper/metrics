package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Метрики Prometheus
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	activeRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	itemsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "items_total",
			Help: "Total number of items in the store",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(activeRequests)
	prometheus.MustRegister(itemsTotal)
}

// Item представляет элемент в хранилище
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Response — общий формат ответа
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// In-memory хранилище
var (
	mu    sync.RWMutex
	items = []Item{
		{ID: 1, Name: "item-1", Value: "value-1"},
		{ID: 2, Name: "item-2", Value: "value-2"},
	}
	nextID = 3
)

// middleware для сбора метрик
func metricsMiddleware(next http.HandlerFunc, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		activeRequests.Inc()
		defer activeRequests.Dec()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		duration := time.Since(start).Seconds()
		// Simulate some variable latency for demo
		_ = rand.Float64()

		httpRequestDuration.WithLabelValues(r.Method, endpoint).Observe(duration)
		httpRequestsTotal.WithLabelValues(r.Method, endpoint, http.StatusText(rw.statusCode)).Inc()
	}
}

// responseWriter оборачивает http.ResponseWriter для перехвата статус-кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// GET /items
func handleGetItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "method not allowed"})
		return
	}
	mu.RLock()
	snapshot := make([]Item, len(items))
	copy(snapshot, items)
	mu.RUnlock()

	writeJSON(w, http.StatusOK, Response{Success: true, Data: snapshot})
}

// POST /items
func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "method not allowed"})
		return
	}
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Message: "invalid request body"})
		return
	}

	mu.Lock()
	item.ID = nextID
	nextID++
	items = append(items, item)
	total := len(items)
	mu.Unlock()

	itemsTotal.Set(float64(total))
	writeJSON(w, http.StatusCreated, Response{Success: true, Data: item})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: "ok"})
}

// GET /ping
func handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, Response{Success: true, Message: "pong"})
}

func main() {
	// Инициализируем начальное значение метрики
	itemsTotal.Set(float64(len(items)))

	mux := http.NewServeMux()

	// Маршруты API
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			metricsMiddleware(handleGetItems, "/items")(w, r)
		case http.MethodPost:
			metricsMiddleware(handleCreateItem, "/items")(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Message: "method not allowed"})
		}
	})
	mux.HandleFunc("/health", metricsMiddleware(handleHealth, "/health"))
	mux.HandleFunc("/ping", metricsMiddleware(handlePing, "/ping"))

	// Эндпоинт Prometheus
	mux.Handle("/metrics", promhttp.Handler())

	log.Println("Starting server on :8080")
	log.Println("  GET  /items   - list all items")
	log.Println("  POST /items   - create new item")
	log.Println("  GET  /health  - health check")
	log.Println("  GET  /ping    - ping")
	log.Println("  GET  /metrics - Prometheus metrics")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
