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
)

func init() {
	prometheus.MustRegister(EventsTotal)
	prometheus.MustRegister(UsersOnline)
}

func Handle(router *mux.Router) {
	router.Handle("/metrics", promhttp.Handler())
}
