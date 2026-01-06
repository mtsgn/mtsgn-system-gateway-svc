package main

import (
	"fmt"
	"log"
	"net/http"
)

// This is a simple test microservice to use with the API Gateway
// Run this to test the gateway functionality

func main() {
	http.HandleFunc("/api/users", handleUsers)
	http.HandleFunc("/api/orders", handleOrders)
	http.HandleFunc("/api/products", handleProducts)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := ":8081"
	log.Printf("Test microservice running on %s", port)
	log.Printf("Available endpoints:")
	log.Printf("  GET /api/users")
	log.Printf("  GET /api/orders")
	log.Printf("  GET /api/products")
	log.Printf("  GET /health")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	log.Printf("Headers: %v", r.Header)

	// Get user info from headers added by gateway
	userID := r.Header.Get("X-User-ID")
	username := r.Header.Get("X-Username")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := fmt.Sprintf(`{
  "service": "user-service",
  "method": "%s",
  "user_id": "%s",
  "username": "%s",
  "message": "This is a test response from user service"
}`, r.Method, userID, username)

	w.Write([]byte(response))
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	log.Printf("Headers: %v", r.Header)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
  "service": "order-service",
  "method": "` + r.Method + `",
  "message": "This is a test response from order service"
}`))
}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	log.Printf("Headers: %v", r.Header)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
  "service": "product-service",
  "method": "` + r.Method + `",
  "message": "This is a test response from product service"
}`))
}
