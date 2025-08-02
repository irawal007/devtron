package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// queryHandler is the HTTP handler for executing a user-provided SQL query.
// It is designed to be a read-only endpoint.
func queryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		log.Printf("Failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	query := string(body)
	if query == "" {
		http.Error(w, "Query cannot be empty", http.StatusBadRequest)
		return
	}

	// Execute the user-provided query.
	// IMPORTANT: This endpoint is intended for read-only queries.
	// The database user should have read-only permissions to mitigate the risk of malicious queries.
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to execute query", http.StatusInternalServerError)
		log.Printf("Failed to execute query: %v", err)
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, "Failed to get columns", http.StatusInternalServerError)
		log.Printf("Failed to get columns: %v", err)
		return
	}

	// Fetch rows and convert them to a slice of maps.
	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			log.Printf("Failed to scan row: %v", err)
			return
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert byte slices to strings for better JSON representation.
			b, ok := val.([]byte)
			if ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode results to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode results to JSON: %v", err)
	}
}
