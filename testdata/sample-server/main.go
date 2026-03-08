package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	items   = []Item{}
	mu      sync.RWMutex
	nextID  = 1
	startAt = time.Now()
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	// Seed some data
	for i := 1; i <= 20; i++ {
		items = append(items, Item{ID: i, Name: "item-" + strconv.Itoa(i)})
	}
	nextID = 21

	http.HandleFunc("/items", handleItems)
	http.HandleFunc("/items/", handleItemByID)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleItems handles GET /items and POST /items.
// BUG: GET ignores limit and offset query params — always returns all items.
func handleItems(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		defer mu.RUnlock()

		// BUG: limit and offset are parsed but never applied
		_ = r.URL.Query().Get("limit")
		_ = r.URL.Query().Get("offset")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		var item Item
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		item.ID = nextID
		nextID++
		items = append(items, item)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleItemByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/items/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, item := range items {
		if item.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(item)
			return
		}
	}

	http.Error(w, "not found", http.StatusNotFound)
}
