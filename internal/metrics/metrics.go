package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OrdersCreatedTotal is a counter that tracks the total number of orders created successfully.
var OrdersCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "orderly_orders_created_total",
	Help: "Total number of orders created successfully.",
})

// OrdersFailedTotal is a counter that tracks the total number of order creations that failed.
var OrdersFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "orderly_orders_failed_total",
	Help: "Total number of order creations that failed.",
})