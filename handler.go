package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cloudflare/migp-go/pkg/migp"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Text     string `json:"text"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type MIGPResponse struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Status   string `json:"status"`
	Metadata string `json:"metadata,omitempty"`
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
	var username, password string
	var response MIGPResponse

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
		// retrieve the config from the server
		var cfg migp.Config
		targetURL := "https://migp.cloudflare.com"
		resp, err := http.Get(targetURL + "/config")
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Unable to retrieve MIGP config from target", http.StatusInternalServerError)
			return
		}
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Unable to retrieve MIGP config from target %q: status code %d", targetURL, resp.StatusCode)
			http.Error(w, "Unable to retrieve MIGP config from target", http.StatusInternalServerError)
			return
		}
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&cfg); err != nil {
			log.Fatal(err)
		}

		if status, metadata, err := migp.Query(cfg, targetURL+"/evaluate", []byte(username), []byte(password)); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else {
			response = MIGPResponse{
				Username: string(username),
				Password: string(password),
				Status:   status.String(),
				Metadata: string(metadata),
			}
		}
	} else {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
		username = "n/a"
		password = "n/a"
		response = MIGPResponse{
			Username: username,
			Password: password,
			Status:   "error",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	out, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(out)
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
