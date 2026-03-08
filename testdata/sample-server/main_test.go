package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupTestServer() {
	items = []Item{
		{ID: 1, Name: "alpha"},
		{ID: 2, Name: "bravo"},
		{ID: 3, Name: "charlie"},
		{ID: 4, Name: "delta"},
		{ID: 5, Name: "echo"},
		{ID: 6, Name: "foxtrot"},
		{ID: 7, Name: "golf"},
		{ID: 8, Name: "hotel"},
		{ID: 9, Name: "india"},
		{ID: 10, Name: "juliet"},
	}
	nextID = 11
}

func TestListItems(t *testing.T) {
	setupTestServer()

	req := httptest.NewRequest("GET", "/items", nil)
	w := httptest.NewRecorder()
	handleItems(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got []Item
	json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 10 {
		t.Fatalf("expected 10 items, got %d", len(got))
	}
}

func TestListPagination(t *testing.T) {
	setupTestServer()

	// Test limit
	req := httptest.NewRequest("GET", "/items?limit=3", nil)
	w := httptest.NewRecorder()
	handleItems(w, req)

	var got []Item
	json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 3 {
		t.Fatalf("limit=3: expected 3 items, got %d", len(got))
	}

	// Test offset
	req = httptest.NewRequest("GET", "/items?limit=3&offset=2", nil)
	w = httptest.NewRecorder()
	handleItems(w, req)

	got = nil
	json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 3 {
		t.Fatalf("limit=3&offset=2: expected 3 items, got %d", len(got))
	}
	if got[0].Name != "charlie" {
		t.Fatalf("expected first item 'charlie', got '%s'", got[0].Name)
	}

	// Test offset past end
	req = httptest.NewRequest("GET", "/items?limit=5&offset=8", nil)
	w = httptest.NewRecorder()
	handleItems(w, req)

	got = nil
	json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 2 {
		t.Fatalf("limit=5&offset=8: expected 2 items, got %d", len(got))
	}
}

func TestCreateItem(t *testing.T) {
	setupTestServer()

	body := strings.NewReader(`{"name":"kilo"}`)
	req := httptest.NewRequest("POST", "/items", body)
	w := httptest.NewRecorder()
	handleItems(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var got Item
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "kilo" {
		t.Fatalf("expected name 'kilo', got '%s'", got.Name)
	}
	if got.ID != 11 {
		t.Fatalf("expected id 11, got %d", got.ID)
	}
}

func TestGetItemByID(t *testing.T) {
	setupTestServer()

	req := httptest.NewRequest("GET", "/items/3", nil)
	w := httptest.NewRecorder()
	handleItemByID(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got Item
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "charlie" {
		t.Fatalf("expected 'charlie', got '%s'", got.Name)
	}
}

func TestGetItemNotFound(t *testing.T) {
	setupTestServer()

	req := httptest.NewRequest("GET", "/items/999", nil)
	w := httptest.NewRecorder()
	handleItemByID(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
