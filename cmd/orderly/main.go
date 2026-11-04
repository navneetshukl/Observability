package main

import (
	"log"

	"orderly/internal/handlers"
	"orderly/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	router := gin.Default()

	// Apply metrics middleware to all routes
	router.Use(middleware.MetricsMiddleware())

	router.POST("/orders", handlers.CreateOrder)
	router.GET("/orders/:id", handlers.GetOrder)
	router.GET("/health", handlers.Health)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	log.Println("listening on :8080")
	log.Fatal(router.Run(":8080"))
}