package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cloudflare/circl/oprf"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Text     string `json:"text"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	message := "Azure Func triggered..."

	if r.Method == http.MethodGet {
		message = "HTTP GET triggered function executed successfully"
	} else if r.Method == http.MethodPost {
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

func migpQueryHandler(w http.ResponseWriter, r *http.Request) {
	var message, username, password string
	suite := oprf.SuiteP256
	client := oprf.NewClient(suite)
	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		decodedBody, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			http.Error(w, "Unable to decode base64 body", http.StatusBadRequest)
			return
		}
		usernameLength := int(decodedBody[0])<<8 | int(decodedBody[1])
		username = string(decodedBody[2 : 2+usernameLength])
		passwordLength := int(decodedBody[2+usernameLength])<<8 | int(decodedBody[3+usernameLength])
		password = string(decodedBody[4+usernameLength : 4+usernameLength+passwordLength])
		message = string(decodedBody)

		inputs := [][]byte{[]byte(username), []byte(password)}
		client.Blind(inputs)
	} else {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		message = "Unsupported method"
		username = "n/a"
		password = "n/a"
	}

	response := Response{Text: message, Username: username, Password: password}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func main() {
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	http.HandleFunc("/api/HttpTrigger1", helloHandler)
	http.HandleFunc("/api/migpQuery", migpQueryHandler)
	log.Printf("About to listen on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
