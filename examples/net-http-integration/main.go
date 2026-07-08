package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	onceo "github.com/HalxDocs/onceo-core"
	"github.com/HalxDocs/onceo-core/providers/paystack"
)

var store = onceo.NewMemoryStore()

func main() {
	http.HandleFunc("/webhook/paystack", handlePaystack)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func handlePaystack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	provider := &paystack.Provider{
		SecretKey: "sk_test_abc123",
	}

	event, err := onceo.Process(r.Context(), provider, store, r.Header, body)
	if err != nil {
		if errors.Is(err, onceo.ErrInvalidSignature) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, onceo.ErrDuplicateEvent) {
			w.WriteHeader(http.StatusOK)
			return
		}
		log.Printf("processing error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}
