package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Deployment represents a deployment with its associated data for the API response.
type Deployment struct {
	ID              int       `json:"id"`
	AppName         string    `json:"appName"`
	EnvironmentName string    `json:"environmentName"`
	Status          string    `json:"status"`
	StartedOn       time.Time `json:"startedOn"`
	FinishedOn      time.Time `json:"finishedOn"`
	TriggeredBy     int       `json:"triggeredBy"`
	Image           string    `json:"image"`
	WorkflowType    string    `json:"workflowType"`
	ImageDigest     string    `json:"imageDigest"`
}

// getDeployments fetches a list of deployments from the database.
func getDeployments(w http.ResponseWriter, r *http.Request) ([]Deployment, error) {
	// Query to fetch deployments
	rows, err := db.Query(`
		SELECT
			cwr.id,
			a.app_name,
			e.environment_name,
			cwr.status,
			cwr.started_on,
			cwr.finished_on,
			cwr.triggered_by,
			cia.image,
			cwr.workflow_type,
			cia.image_digest
		FROM
			cd_workflow_runner cwr
		JOIN
			cd_workflow cw ON cwr.cd_workflow_id = cw.id
		JOIN
			pipeline p ON cw.pipeline_id = p.id
		JOIN
			app a ON p.app_id = a.id
		JOIN
			environment e ON p.environment_id = e.id
		JOIN
			ci_artifact cia ON cw.ci_artifact_id = cia.id
		ORDER BY
			cwr.started_on DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.AppName, &d.EnvironmentName, &d.Status, &d.StartedOn, &d.FinishedOn, &d.TriggeredBy, &d.Image, &d.WorkflowType, &d.ImageDigest); err != nil {
			return nil, err
		}
		deployments = append(deployments, d)
	}
	return deployments, nil
}

// getDeploymentsHandler is the HTTP handler for fetching deployments.
func getDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	deployments, err := getDeployments(w, r)
	if err != nil {
		http.Error(w, "Failed to query deployments", http.StatusInternalServerError)
		log.Printf("Failed to query deployments: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deployments); err != nil {
		http.Error(w, "Failed to encode deployments to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode deployments to JSON: %v", err)
	}
}

// exportDeploymentsHandler is the HTTP handler for exporting deployments to CSV.
func exportDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	deployments, err := getDeployments(w, r)
	if err != nil {
		http.Error(w, "Failed to query deployments", http.StatusInternalServerError)
		log.Printf("Failed to query deployments: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=deployments.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "AppName", "EnvironmentName", "Status", "StartedOn", "FinishedOn", "TriggeredBy", "Image", "WorkflowType", "ImageDigest"}
	if err := writer.Write(header); err != nil {
		log.Printf("Failed to write CSV header: %v", err)
		return
	}

	// Write rows
	for _, d := range deployments {
		row := []string{
			strconv.Itoa(d.ID),
			d.AppName,
			d.EnvironmentName,
			d.Status,
			d.StartedOn.String(),
			d.FinishedOn.String(),
			strconv.Itoa(d.TriggeredBy),
			d.Image,
			d.WorkflowType,
			d.ImageDigest,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Failed to write CSV row: %v", err)
			return
		}
	}
}

// exportMonthlyDeploymentsHandler is the HTTP handler for exporting monthly deployments to CSV.
func exportMonthlyDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		http.Error(w, "Invalid month parameter", http.StatusBadRequest)
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "Invalid year parameter", http.StatusBadRequest)
		return
	}

	// Query to fetch monthly deployments
	rows, err := db.Query(`
		SELECT
			cwr.id,
			a.app_name,
			e.environment_name,
			cwr.status,
			cwr.started_on,
			cwr.finished_on,
			cwr.triggered_by,
			cia.image
		FROM
			cd_workflow_runner cwr
		JOIN
			cd_workflow cw ON cwr.cd_workflow_id = cw.id
		JOIN
			pipeline p ON cw.pipeline_id = p.id
		JOIN
			app a ON p.app_id = a.id
		JOIN
			environment e ON p.environment_id = e.id
		JOIN
			ci_artifact cia ON cw.ci_artifact_id = cia.id
		WHERE
			EXTRACT(MONTH FROM cwr.started_on) = $1
			AND EXTRACT(YEAR FROM cwr.started_on) = $2
		ORDER BY
			cwr.started_on DESC;
	`, month, year)
	if err != nil {
		http.Error(w, "Failed to query monthly deployments", http.StatusInternalServerError)
		log.Printf("Failed to query monthly deployments: %v", err)
		return
	}
	defer rows.Close()

	var deployments []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.AppName, &d.EnvironmentName, &d.Status, &d.StartedOn, &d.FinishedOn, &d.TriggeredBy, &d.Image); err != nil {
			http.Error(w, "Failed to scan deployment data", http.StatusInternalServerError)
			log.Printf("Failed to scan deployment data: %v", err)
			return
		}
		deployments = append(deployments, d)
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=deployments-"+yearStr+"-"+monthStr+".csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "AppName", "EnvironmentName", "Status", "StartedOn", "FinishedOn", "TriggeredBy", "Image"}
	if err := writer.Write(header); err != nil {
		log.Printf("Failed to write CSV header: %v", err)
		return
	}

	// Write rows
	for _, d := range deployments {
		row := []string{
			strconv.Itoa(d.ID),
			d.AppName,
			d.EnvironmentName,
			d.Status,
			d.StartedOn.String(),
			d.FinishedOn.String(),
			strconv.Itoa(d.TriggeredBy),
			d.Image,
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Failed to write CSV row: %v", err)
			return
		}
	}
}
