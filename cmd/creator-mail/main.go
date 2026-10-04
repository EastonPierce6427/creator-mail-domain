package main

import (
	"encoding/json"
	"log"
	"net/http"

	"creator-mail-domain/internal/creator"
)

func main() {
	client, err := creator.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/deliveries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()
		var input creator.Delivery
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := creator.Deliver(r.Context(), client, input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	log.Println("creator mail listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
