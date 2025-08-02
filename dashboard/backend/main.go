package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

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
	// It is recommended to use environment variables for connection details in a real application
	// For example:
	// dbUser := os.Getenv("READONLY_DB_USER")
	// dbPassword := os.Getenv("READONLY_DB_PASSWORD")
	// dbName := os.Getenv("DB_NAME")
	// dbHost := os.Getenv("DB_HOST")
	// dbPort := os.Getenv("DB_PORT")

	// Using placeholder values for now.
	// The user should provide the actual credentials for the read-only user.
	dbUser := "readonly_user"
	dbPassword := "readonly_password"
	dbName := "orchestrator" // As discovered from the main app's config
	dbHost := "127.0.0.1"      // Assuming the dashboard runs in the same cluster
	dbPort := "5432"

	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		dbUser, dbPassword, dbName, dbHost, dbPort)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	// not closing db connection here, as it will be used by handlers

	if err = db.Ping(); err != nil {
		log.Printf("Initial database connection failed: %v", err)
	} else {
		log.Println("Successfully connected to the database")
	}

	// Serve frontend static files
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/users", getUsersHandler)
	http.HandleFunc("/api/users/export", exportUsersHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
