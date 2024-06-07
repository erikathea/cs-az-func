package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Text string `json:"text"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	message := "Azure Func triggered..."

	if r.Method == http.MethodGet {
		message = "HTTP GET triggered function executed successfully"
	} else if r.Method == http.MethodPost {
		// Handle POST request
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			message = "Bad Request"
		}

		if req.Name != "" {
			message = fmt.Sprintf("Hello, %s!", req.Name)
		}
	} else {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		message = "Unsupported method"
	}

	response := Response{Text: message}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	http.HandleFunc("/api/HttpTrigger1", helloHandler)
	log.Printf("About to listen on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
