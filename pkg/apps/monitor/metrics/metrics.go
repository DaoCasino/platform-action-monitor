package metrics

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	EventsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "events_total",
		})

	UsersOnline = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_online",
		})

	ConnectionPoolSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connection_pool_size",
		})

	SessionsAmount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sessions_amount",
		})

	EventDataDecodingTimeMs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "event_data_decoding_time_ms",
			Buckets: []float64{5, 10, 20, 50, 100},
		})

	NotifyProcessingTimeMs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "notify_processing_time_ms",
			Buckets: []float64{5, 10, 20, 50, 100},
		})
)

func init() {
	prometheus.MustRegister(EventsTotal)
	prometheus.MustRegister(UsersOnline)
	prometheus.MustRegister(ConnectionPoolSize)
	prometheus.MustRegister(SessionsAmount)
	prometheus.MustRegister(EventDataDecodingTimeMs)
	prometheus.MustRegister(NotifyProcessingTimeMs)
}

func Handle(router *mux.Router) {
	router.Handle("/metrics", promhttp.Handler())
}
