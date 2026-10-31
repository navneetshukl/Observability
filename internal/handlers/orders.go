package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"orderly/internal/metrics"
)

// Order represents an order in the system.
type Order struct {
	ID     int     `json:"id"`
	Item   string  `json:"item"`
	Amount float64 `json:"amount"`
}

var (
	mu     sync.Mutex
	orders = map[int]Order{}
	nextID = 1
)

// CreateOrder handles POST /orders - creates an order with artificial latency and ~5% failure rate.
func CreateOrder(c *gin.Context) {
	// Artificial latency: 10-300 ms
	time.Sleep(time.Duration(10+rand.Intn(290)) * time.Millisecond)

	// ~5% random failures
	if rand.Intn(100) < 5 {
		metrics.OrdersFailedTotal.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	var o Order
	if err := c.ShouldBindJSON(&o); err != nil {
		metrics.OrdersFailedTotal.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	mu.Lock()
	o.ID = nextID
	nextID++
	orders[o.ID] = o
	mu.Unlock()

	metrics.OrdersCreatedTotal.Inc()
	c.JSON(http.StatusCreated, o)
}

// GetOrder handles GET /orders/:id - returns the order or 404.
func GetOrder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}

	mu.Lock()
	o, ok := orders[id]
	mu.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, o)
}

// Health handles GET /health - returns 200 OK.
func Health(c *gin.Context) {
	c.Status(http.StatusOK)
}