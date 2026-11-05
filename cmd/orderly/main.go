package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"orderly/internal/database"
	"orderly/internal/handlers"
	"orderly/internal/metrics"
	"orderly/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const version = "1.0.0"

func main() {
	startTime := time.Now()

	// Initialize metrics with build info
	metrics.InitBuildInfo(version)
	metrics.InitProcessStartTime(startTime)

	// Initialize database
	db, err := database.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize handlers with database
	handlers.InitDB(db)

	// Start background goroutine to update pool metrics
	ctx, cancel := context.WithCancel(context.Background())
	go updatePoolMetrics(ctx, db)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		db.Close()
		os.Exit(0)
	}()

	router := gin.Default()

	// Apply metrics middleware to all routes
	router.Use(middleware.MetricsMiddleware())

	router.POST("/orders", handlers.CreateOrder)
	router.GET("/orders/:id", handlers.GetOrder)
	router.GET("/health", handlers.Health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Failure experiment endpoints (temporary, for testing)
	setupFailureEndpoints(router)

	log.Printf("listening on :8080 (version: %s, go: %s)", version, runtime.Version())
	log.Fatal(router.Run(":8080"))
}

// updatePoolMetrics periodically updates the database pool metrics.
func updatePoolMetrics(ctx context.Context, db *database.DB) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			db.UpdatePoolMetrics()
		}
	}
}

// setupFailureEndpoints adds temporary endpoints for failure experiments.
// These should be removed in production.
func setupFailureEndpoints(router *gin.Engine) {
	// Goroutine leak endpoint - starts N goroutines that block forever
	router.POST("/debug/leak", func(c *gin.Context) {
		n := 10
		if val := c.Query("n"); val != "" {
			// Ignore parse error, use default
		}
		for i := 0; i < n; i++ {
			go func() {
				// This goroutine blocks forever, simulating a leak
				select {}
			}()
		}
		c.JSON(200, gin.H{"message": "started leak goroutines", "count": n})
	})

	// Toggle slow database mode
	router.POST("/debug/slow-db", func(c *gin.Context) {
		// Note: This requires restart to take effect since it's read at startup
		// In a real scenario, you'd use a shared atomic bool
		c.JSON(200, gin.H{"message": "slow DB mode is controlled by SLOW_DB env var at startup"})
	})
}