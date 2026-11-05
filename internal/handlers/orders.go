package handlers

import (
	"context"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"orderly/internal/database"
	"orderly/internal/metrics"

	"github.com/gin-gonic/gin"
)

var db *database.DB

// InitDB initializes the database connection for handlers.
func InitDB(database *database.DB) {
	db = database
}

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

	var o database.Order
	if err := c.ShouldBindJSON(&o); err != nil {
		metrics.OrdersFailedTotal.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	order, err := db.CreateOrder(ctx, o.Item, o.Amount)
	if err != nil {
		metrics.OrdersFailedTotal.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	metrics.OrdersCreatedTotal.Inc()
	c.JSON(http.StatusCreated, order)
}

// GetOrder handles GET /orders/:id - returns the order or 404.
func GetOrder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		metrics.OrdersFailedTotal.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	order, err := db.GetOrder(ctx, id)
	if err != nil {
		if err == database.ErrOrderNotFound {
			metrics.OrdersFailedTotal.Inc()
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		metrics.OrdersFailedTotal.Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	// Note: GetOrder doesn't increment OrdersCreatedTotal - only successful creates do
	c.JSON(http.StatusOK, order)
}

// Health handles GET /health - returns 200 OK.
func Health(c *gin.Context) {
	c.Status(http.StatusOK)
}
