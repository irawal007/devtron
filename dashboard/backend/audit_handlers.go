package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// AuditLog represents a single audit log entry for the API response.
type AuditLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"userId"`
	UserEmail string    `json:"userEmail"`
	ClientIP  string    `json:"clientIp"`
	Timestamp time.Time `json:"timestamp"`
}

// TriggerLog represents a single trigger event.
type TriggerLog struct {
	TriggeredBy string    `json:"triggeredBy"`
	AppName     string    `json:"appName"`
	Pipeline    string    `json:"pipeline"`
	Timestamp   time.Time `json:"timestamp"`
}

// getAuditsHandler is the HTTP handler for fetching audit logs.
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

func getTriggerAuditsHandler(w http.ResponseWriter, r *http.Request) {
	// Query to fetch trigger audit data
	rows, err := db.Query(`
		SELECT
			u.email_id,
			a.app_name,
			p.pipeline_name,
			cwr.started_on
		FROM
			cd_workflow_runner cwr
		JOIN
			users u ON cwr.triggered_by = u.id
		JOIN
			cd_workflow cw ON cwr.cd_workflow_id = cw.id
		JOIN
			pipeline p ON cw.pipeline_id = p.id
		JOIN
			app a ON p.app_id = a.id
		ORDER BY
			cwr.started_on DESC;
	`)
	if err != nil {
		http.Error(w, "Failed to query trigger audits", http.StatusInternalServerError)
		log.Printf("Failed to query trigger audits: %v", err)
		return
	}
	defer rows.Close()

	var triggerAudits []TriggerLog
	for rows.Next() {
		var ta TriggerLog
		if err := rows.Scan(&ta.TriggeredBy, &ta.AppName, &ta.Pipeline, &ta.Timestamp); err != nil {
			http.Error(w, "Failed to scan trigger audit data", http.StatusInternalServerError)
			log.Printf("Failed to scan trigger audit data: %v", err)
			return
		}
		triggerAudits = append(triggerAudits, ta)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(triggerAudits); err != nil {
		http.Error(w, "Failed to encode trigger audits to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode trigger audits to JSON: %v", err)
	}
}
