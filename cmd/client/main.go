// Client for testing the Orderly HTTP handlers
//
// This makes real HTTP requests to the running server (cmd/test/main.go)
// to exercise all three handlers with actual operations:
//   - POST /orders     -> handlers.CreateOrder (creates order, 10-300ms latency, ~5% random failure)
//   - GET  /orders/:id -> handlers.GetOrder    (retrieves order from in-memory map)
//   - GET  /health     -> handlers.Health      (returns 200 OK)
//
// Run server first:  go run ./cmd/test/main.go
// Then run client:   go run ./cmd/client/main.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Order struct {
	ID     int     `json:"id"`
	Item   string  `json:"item"`
	Amount float64 `json:"amount"`
}

func main() {
	for {
		baseURL := "http://localhost:8080"
		client := &http.Client{Timeout: 5 * time.Second}

		// 1. Create an order
		fmt.Println("=== Creating order ===")
		createReq := Order{Item: "Laptop", Amount: 999.99}
		body, _ := json.Marshal(createReq)

		resp, err := client.Post(baseURL+"/orders", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Printf("Error creating order: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var created Order
		json.NewDecoder(resp.Body).Decode(&created)
		fmt.Printf("Created: %+v (status: %d)\n", created, resp.StatusCode)

		// 2. Get the order
		fmt.Println("\n=== Getting order ===")
		resp, err = client.Get(baseURL + "/orders/" + fmt.Sprint(created.ID))
		if err != nil {
			fmt.Printf("Error getting order: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var got Order
		json.NewDecoder(resp.Body).Decode(&got)
		fmt.Printf("Retrieved: %+v (status: %d)\n", got, resp.StatusCode)

		// 3. Health check
		fmt.Println("\n=== Health check ===")
		resp, err = client.Get(baseURL + "/health")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Printf("Health status: %d\n", resp.StatusCode)
		resp.Body.Close()

		// 4. Create another order to show counter increments
		fmt.Println("\n=== Creating second order ===")
		createReq2 := Order{Item: "Mouse", Amount: 29.99}
		body2, _ := json.Marshal(createReq2)

		resp, err = client.Post(baseURL+"/orders", "application/json", bytes.NewBuffer(body2))
		if err != nil {
			fmt.Printf("Error creating order: %v\n", err)
			return
		}
		defer resp.Body.Close()

		var created2 Order
		json.NewDecoder(resp.Body).Decode(&created2)
		fmt.Printf("Created: %+v (status: %d)\n", created2, resp.StatusCode)
		time.Sleep(10 * time.Second)
	}
}
