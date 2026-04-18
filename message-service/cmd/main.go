package main

import (
	"fmt"
	"message-service/internal/handlers"
	"message-service/pkg/db"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})
	http.HandleFunc("/send", handlers.SendMessage)

	db.InitDB()

	fmt.Println("Server running on port: 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server failed to start")
	}
}