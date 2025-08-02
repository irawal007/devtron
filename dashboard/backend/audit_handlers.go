package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// AuditLog represents a single audit log entry.
type AuditLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"userId"`
	UserEmail string    `json:"userEmail"`
	ClientIP  string    `json:"clientIp"`
	Timestamp time.Time `json:"timestamp"`
}

func getAuditsHandler(w http.ResponseWriter, r *http.Request) {
	// Query to fetch user audit data
	rows, err := db.Query(`
		SELECT
			ua.id,
			ua.user_id,
			u.email_id,
			ua.client_ip,
			ua.created_on
		FROM
			user_audit ua
		JOIN
			users u ON ua.user_id = u.id
		ORDER BY
			ua.created_on DESC;
	`)
	if err != nil {
		http.Error(w, "Failed to query audits", http.StatusInternalServerError)
		log.Printf("Failed to query audits: %v", err)
		return
	}
	defer rows.Close()

	var audits []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.UserID, &a.UserEmail, &a.ClientIP, &a.Timestamp); err != nil {
			http.Error(w, "Failed to scan audit data", http.StatusInternalServerError)
			log.Printf("Failed to scan audit data: %v", err)
			return
		}
		audits = append(audits, a)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(audits); err != nil {
		http.Error(w, "Failed to encode audits to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode audits to JSON: %v", err)
	}
}
