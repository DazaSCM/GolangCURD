package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

const (
	dbDriver = "mysql"
	dbUser   = "root"
	dbPass   = "root"
	dbName   = "gocrud_app"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/users", getAllUsersHandler).Methods("GET")
	r.HandleFunc("/user", createUserHandler).Methods("POST")
	r.HandleFunc("/user/{id}", getUserHandler).Methods("GET")
	r.HandleFunc("/user/{id}", updateUserHandler).Methods("PUT")
	r.HandleFunc("/user/{id}", deleteUserHandler).Methods("DELETE")

	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func dbConnection() (*sql.DB, error) {
	return sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
}

// Validate checks if the User struct contains valid data
func (u *User) Validate() error {
	if u.Name == "" {
		return fmt.Errorf("name is required")
	}
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !isValidEmail(u.Email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	db, err := dbConnection()
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	user, err := GetAllUsers(db)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := dbConnection()
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate user data
	if err := user.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := CreateUser(db, user.Name, user.Email); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "User created successfully")
}

func CreateUser(db *sql.DB, name, email string) error {
	_, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", name, email)
	return err
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := dbConnection()
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	userID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := GetUser(db, userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func GetUser(db *sql.DB, id int) (*User, error) {
	row := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id)
	user := &User{}
	if err := row.Scan(&user.ID, &user.Name, &user.Email); err != nil {
		return nil, err
	}
	return user, nil
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := dbConnection()
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	userID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Validate user data
	if err := user.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := UpdateUser(db, userID, user.Name, user.Email); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "User updated successfully")
}

func UpdateUser(db *sql.DB, id int, name, email string) error {
	_, err := db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", name, email, id)
	return err
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	db, err := dbConnection()
	if err != nil {
		http.Error(w, "Failed to connect to the database", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	userID, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := DeleteUser(db, userID); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "User deleted successfully")
}

func DeleteUser(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func parseID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	return strconv.Atoi(vars["id"])
}

// isValidEmail validates an email string using a regex pattern
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
