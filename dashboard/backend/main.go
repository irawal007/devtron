package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// db is the global database connection pool.
var db *sql.DB

// healthHandler checks the health of the application, including the database connection.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if db != nil {
		if err := db.Ping(); err != nil {
			http.Error(w, "Database connection failed", http.StatusInternalServerError)
			log.Printf("Database connection failed: %v", err)
			return
		}
	} else {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		log.Printf("Database not initialized")
		return
	}

	fmt.Fprintf(w, "OK")
}

func main() {
	// Read database credentials from environment variables.
	dbUser := os.Getenv("PG_USER")
	dbPassword := os.Getenv("PG_PASSWORD")
	dbName := os.Getenv("PG_DATABASE")
	dbHost := os.Getenv("PG_ADDR")
	dbPort := os.Getenv("PG_PORT")

	if dbUser == "" || dbPassword == "" || dbName == "" || dbHost == "" || dbPort == "" {
		log.Fatal("Database environment variables are not set. Please set PG_USER, PG_PASSWORD, PG_DATABASE, PG_ADDR, and PG_PORT.")
	}


	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		dbUser, dbPassword, dbName, dbHost, dbPort)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	// not closing db connection here, as it will be used by handlers

	// Ping the database to verify the connection on startup.
	if err = db.Ping(); err != nil {
		log.Printf("Initial database connection failed: %v", err)
	} else {
		log.Println("Successfully connected to the database")
	}

	// Serve frontend static files from the 'frontend' directory.
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// Register API handlers.
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/users", getUsersHandler)
	http.HandleFunc("/api/users/export", exportUsersHandler)
	http.HandleFunc("/api/deployments", getDeploymentsHandler)
	http.HandleFunc("/api/deployments/export", exportDeploymentsHandler)
	http.HandleFunc("/api/deployments/export/monthly", exportMonthlyDeploymentsHandler)
	http.HandleFunc("/api/applications", getApplicationsHandler)
	http.HandleFunc("/api/applications/export", exportApplicationsHandler)
	http.HandleFunc("/api/audits", getAuditsHandler)
	http.HandleFunc("/api/query", queryHandler)
	http.HandleFunc("/api/applications/virtual-service-usage", getVirtualServiceUsageHandler)
	http.HandleFunc("/api/applications/not-deployed", getAppsNotDeployedHandler)
	http.HandleFunc("/api/ci/build-time", getBuildTimeAnalyticsHandler)
	http.HandleFunc("/api/applications/chart-version-usage", getChartVersionUsageHandler)
	http.HandleFunc("/api/scoped-variable-usage", getScopedVariableUsageHandler)
	http.HandleFunc("/api/audits/triggers", getTriggerAuditsHandler)
	http.HandleFunc("/api/applications/deployed-branch", getDeployedBranchHandler)
	http.HandleFunc("/api/pipelines/no-image-scan", getNoImageScanReportHandler)
	http.HandleFunc("/api/applications/trivy-scan/export", exportTrivyScanResultHandler)
	http.HandleFunc("/api/cluster/dump-secrets-configmaps", dumpSecretsAndConfigMapsHandler)


	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
