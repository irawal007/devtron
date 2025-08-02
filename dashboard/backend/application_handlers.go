package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Application represents an application with its version history for the API response.
type Application struct {
	ID             int          `json:"id"`
	AppName        string       `json:"appName"`
	VersionHistory []AppVersion `json:"versionHistory"`
}

// AppVersion represents a specific version of an application for the API response.
type AppVersion struct {
	InstalledAppVersionID int       `json:"installedAppVersionId"`
	StartedOn             time.Time `json:"startedOn"`
	FinishedOn            time.Time `json:"finishedOn"`
	Status                string    `json:"status"`
}

// getApplications fetches a list of applications with their version history from the database.
func getApplications(w http.ResponseWriter, r *http.Request) ([]Application, error) {
	// Query to fetch applications and their version history
	rows, err := db.Query(`
		SELECT
			a.id,
			a.app_name,
			iavh.id,
			iavh.started_on,
			iavh.finished_on,
			iavh.status
		FROM
			app a
		JOIN
			installed_apps ia ON a.id = ia.app_id
		JOIN
			installed_app_version_history iavh ON ia.id = iavh.installed_app_id
		ORDER BY
			a.app_name, iavh.started_on DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appsMap := make(map[int]Application)
	for rows.Next() {
		var appID int
		var appName string
		var version AppVersion
		if err := rows.Scan(&appID, &appName, &version.InstalledAppVersionID, &version.StartedOn, &version.FinishedOn, &version.Status); err != nil {
			return nil, err
		}

		app, ok := appsMap[appID]
		if !ok {
			app = Application{
				ID:      appID,
				AppName: appName,
			}
		}
		app.VersionHistory = append(app.VersionHistory, version)
		appsMap[appID] = app
	}

	var apps []Application
	for _, app := range appsMap {
		apps = append(apps, app)
	}
	return apps, nil
}

// getApplicationsHandler is the HTTP handler for fetching applications.
func getApplicationsHandler(w http.ResponseWriter, r *http.Request) {
	apps, err := getApplications(w, r)
	if err != nil {
		http.Error(w, "Failed to query applications", http.StatusInternalServerError)
		log.Printf("Failed to query applications: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		http.Error(w, "Failed to encode applications to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode applications to JSON: %v", err)
	}
}

// exportApplicationsHandler is the HTTP handler for exporting applications to CSV.
func exportApplicationsHandler(w http.ResponseWriter, r *http.Request) {
	apps, err := getApplications(w, r)
	if err != nil {
		http.Error(w, "Failed to query applications", http.StatusInternalServerError)
		log.Printf("Failed to query applications: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=applications.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "AppName", "VersionHistory"}
	if err := writer.Write(header); err != nil {
		log.Printf("Failed to write CSV header: %v", err)
		return
	}

	// Write rows
	for _, app := range apps {
		var versionHistory []string
		for _, v := range app.VersionHistory {
			versionHistory = append(versionHistory, "ID: "+strconv.Itoa(v.InstalledAppVersionID)+", Status: "+v.Status)
		}

		row := []string{
			strconv.Itoa(app.ID),
			app.AppName,
			strings.Join(versionHistory, " | "),
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Failed to write CSV row: %v", err)
			return
		}
	}
}
