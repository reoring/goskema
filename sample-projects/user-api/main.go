package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/reoring/goskema"
	g "github.com/reoring/goskema/dsl"
)

// User represents a user in our system
type User struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Age    int    `json:"age"`
	Active bool   `json:"active"`
}

// UserStore is a simple in-memory store
type UserStore struct {
	mu     sync.RWMutex
	users  map[int]User
	nextID int
}

func NewUserStore() *UserStore {
	return &UserStore{
		users:  make(map[int]User),
		nextID: 1,
	}
}

func (s *UserStore) Create(user User) User {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.ID = s.nextID
	s.nextID++
	s.users[user.ID] = user

	return user
}

func (s *UserStore) GetAll() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

func (s *UserStore) GetByID(id int) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	return user, exists
}

func (s *UserStore) Update(id int, user User) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return false
	}

	user.ID = id
	s.users[id] = user
	return true
}

func (s *UserStore) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return false
	}

	delete(s.users, id)
	return true
}

// Server holds our application state
type Server struct {
	store      *UserStore
	userSchema goskema.Schema[User]
}

func NewServer() *Server {
	// Define User schema using goskema DSL
	userSchema := g.ObjectOf[User]().
		Field("id", g.IntOf[int]()).Default(0). // ID will be set by the server
		Field("name", g.StringOf[string]()).Required().
		Field("email", g.StringOf[string]()).Required(). // TODO: Add email format validation
		Field("age", g.IntOf[int]()).Default(18).        // Default age is 18
		Field("active", g.BoolOf[bool]()).Default(true). // Default to active
		UnknownStrict().                                 // Reject unknown fields
		MustBind()

	return &Server{
		store:      NewUserStore(),
		userSchema: userSchema,
	}
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetUsers(w, r)
	case http.MethodPost:
		s.handleCreateUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetUser(w, r, id)
	case http.MethodPatch:
		s.handlePatchUser(w, r, id)
	case http.MethodDelete:
		s.handleDeleteUser(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetUsers(w http.ResponseWriter, _ *http.Request) {
	users := s.store.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

func (s *Server) handleGetUser(w http.ResponseWriter, _ *http.Request, id int) {
	user, exists := s.store.GetByID(id)
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Parse and validate user data using goskema
	user, err := goskema.ParseFrom(ctx, s.userSchema, goskema.JSONReader(r.Body))
	if err != nil {
		s.handleValidationError(w, err)
		return
	}

	// Create user in store
	createdUser := s.store.Create(user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUser)
}

func (s *Server) handlePatchUser(w http.ResponseWriter, r *http.Request, id int) {
	ctx := context.Background()

	// Check if user exists
	existingUser, exists := s.store.GetByID(id)
	if !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Parse with metadata to track presence information
	decoded, err := goskema.ParseFromWithMeta(ctx, s.userSchema, goskema.JSONReader(r.Body))
	if err != nil {
		s.handleValidationError(w, err)
		return
	}

	// Apply partial update based on presence information
	updatedUser := existingUser

	// Only update fields that were present in the request
	if decoded.Presence["/name"]&goskema.PresenceSeen != 0 {
		updatedUser.Name = decoded.Value.Name
	}
	if decoded.Presence["/email"]&goskema.PresenceSeen != 0 {
		updatedUser.Email = decoded.Value.Email
	}
	if decoded.Presence["/age"]&goskema.PresenceSeen != 0 {
		updatedUser.Age = decoded.Value.Age
	}
	if decoded.Presence["/active"]&goskema.PresenceSeen != 0 {
		updatedUser.Active = decoded.Value.Active
	}

	// Update in store
	s.store.Update(id, updatedUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":           updatedUser,
		"updated_fields": s.getUpdatedFields(decoded.Presence),
	})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, _ *http.Request, id int) {
	if !s.store.Delete(id) {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate JSON Schema from our goskema schema
	jsonSchema, err := s.userSchema.JSONSchema()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate schema: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonSchema)
}

func (s *Server) handleValidationError(w http.ResponseWriter, err error) {
	// Check if it's a goskema validation error
	if issues, ok := goskema.AsIssues(err); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		errorResponse := map[string]interface{}{
			"error":   "Validation failed",
			"issues":  issues,
			"details": make([]map[string]interface{}, len(issues)),
		}

		for i, issue := range issues {
			errorResponse["details"].([]map[string]interface{})[i] = map[string]interface{}{
				"path":    issue.Path,
				"code":    issue.Code,
				"message": issue.Message,
				"hint":    issue.Hint,
			}
		}

		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Handle other errors
	http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusBadRequest)
}

func (s *Server) getUpdatedFields(presence goskema.PresenceMap) []string {
	var updated []string

	fields := []string{"/name", "/email", "/age", "/active"}
	for _, field := range fields {
		if presence[field]&goskema.PresenceSeen != 0 {
			updated = append(updated, strings.TrimPrefix(field, "/"))
		}
	}

	return updated
}

func main() {
	server := NewServer()

	// Add some initial data
	server.store.Create(User{Name: "Taro", Email: "taro@example.com", Age: 30, Active: true})
	server.store.Create(User{Name: "Hanako", Email: "hanako@example.com", Age: 25, Active: true})

	// Setup routes
	http.HandleFunc("/users", server.handleUsers)
	http.HandleFunc("/users/", server.handleUserByID)
	http.HandleFunc("/schema", server.handleSchema)

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Root handler with usage instructions
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "goskema User API Sample",
			"endpoints": map[string]string{
				"GET /users":         "Get all users",
				"POST /users":        "Create a new user",
				"GET /users/{id}":    "Get user by ID",
				"PATCH /users/{id}":  "Partially update user",
				"DELETE /users/{id}": "Delete user",
				"GET /schema":        "Get JSON Schema for User",
				"GET /health":        "Health check",
			},
			"examples": map[string]interface{}{
				"create_user": map[string]interface{}{
					"method": "POST",
					"url":    "/users",
					"body": map[string]interface{}{
						"name":   "Taro",
						"email":  "taro@example.com",
						"age":    30,
						"active": true,
					},
				},
				"partial_update": map[string]interface{}{
					"method": "PATCH",
					"url":    "/users/1",
					"body": map[string]interface{}{
						"name": "Jiro",
					},
					"note": "Only updates the 'name' field, other fields remain unchanged",
				},
			},
		})
	})

	log.Println("🚀 goskema User API server starting on :8080")
	log.Println("📖 Visit http://localhost:8080 for usage instructions")
	log.Println("🔍 Visit http://localhost:8080/schema to see the JSON Schema")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
