package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Received request: %s %s\n", r.Method, r.URL.Path)
		for k, v := range r.Header {
			fmt.Printf("Header: %s=%v\n", k, v)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Hello from mock backend"}`))
	})

	fmt.Println("Mock backend listening on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
