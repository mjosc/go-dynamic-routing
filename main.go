package main

import (
	"fmt"
	"net/http"

	"github.com/mjosc/go-dynamic-routing/pkg/handlers"

	"github.com/go-chi/chi"
)

func main() {

	go func() {
		fmt.Println("mock server started @ localhost:8100")
		r := chi.NewRouter()
		r.Handle("/*", handlers.NewMockService())
		http.ListenAndServe(":8100", r)
	}()

	r := chi.NewRouter()
	services := chi.NewRouter()
	// Add any middleware here to be used across all microservices no matter their configuration
	services.Use()
	r.Method("POST", "/configure", handlers.NewRouteConfigurer(r, services))
	// We'd probably grab this pattern from the config file
	r.Mount("/services", services)

	fmt.Println("server started @ localhost:8080")
	http.ListenAndServe(":8080", r)
}
