package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/lib/pq"
)

// User represents a user with their roles and teams.
type User struct {
	ID       int      `json:"id"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	Team     string   `json:"team"`
	IsActive bool     `json:"isActive"`
}

func getUsers(w http.ResponseWriter, r *http.Request) ([]User, error) {
	// Query to fetch users with their roles and teams
	rows, err := db.Query(`
		SELECT
			u.id,
			u.email_id,
			u.active,
			array_agg(r.role) as roles,
			t.name as team
		FROM
			users u
		LEFT JOIN
			user_roles ur ON u.id = ur.user_id
		LEFT JOIN
			roles r ON ur.role_id = r.id
		LEFT JOIN
			team t ON r.team = t.name
		GROUP BY
			u.id, u.email_id, u.active, t.name
		ORDER BY
			u.id;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usersMap := make(map[int]User)
	for rows.Next() {
		var id int
		var email string
		var isActive bool
		var roles pq.StringArray
		var team sql.NullString

		if err := rows.Scan(&id, &email, &isActive, &roles, &team); err != nil {
			return nil, err
		}

		user, ok := usersMap[id]
		if !ok {
			user = User{
				ID:       id,
				Email:    email,
				IsActive: isActive,
			}
		}

		user.Roles = append(user.Roles, roles...)
		if team.Valid {
			user.Team = team.String
		}
		usersMap[id] = user
	}

	var users []User
	for _, user := range usersMap {
		users = append(users, user)
	}
	return users, nil
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := getUsers(w, r)
	if err != nil {
		http.Error(w, "Failed to query users", http.StatusInternalServerError)
		log.Printf("Failed to query users: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, "Failed to encode users to JSON", http.StatusInternalServerError)
		log.Printf("Failed to encode users to JSON: %v", err)
	}
}

func exportUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := getUsers(w, r)
	if err != nil {
		http.Error(w, "Failed to query users", http.StatusInternalServerError)
		log.Printf("Failed to query users: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=users.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Email", "Roles", "Team", "Active"}
	if err := writer.Write(header); err != nil {
		log.Printf("Failed to write CSV header: %v", err)
		return
	}

	// Write rows
	for _, user := range users {
		row := []string{
			strconv.Itoa(user.ID),
			user.Email,
			pq.Array(user.Roles).String(),
			user.Team,
			strconv.FormatBool(user.IsActive),
		}
		if err := writer.Write(row); err != nil {
			log.Printf("Failed to write CSV row: %v", err)
			return
		}
	}
}
