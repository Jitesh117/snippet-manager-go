package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	database "snippet-manager-go/database"
	"snippet-manager-go/models"
)

type SnippetHandler struct {
	storage *database.PostgresStorage
}

type UserHandler struct {
	storage *database.PostgresStorage
}

func NewUserHandler(storage *database.PostgresStorage) *UserHandler {
	return &UserHandler{storage: storage}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// TODO: Add input validation here

	err = h.storage.CreateUser(&user)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	user.Password = "" // Clear password before sending response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&credentials)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	user, err := h.storage.GetUserByUsername(credentials.Username)
	if err != nil {
		http.Error(w, "Invalid username", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password))
	if err != nil {
		log.Printf("Password comparison failed: %v", err)
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	// TODO: Generate and return JWT token here

	// Clear password before sending response
	user.Password = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func NewSnippetHandler(storage *database.PostgresStorage) *SnippetHandler {
	return &SnippetHandler{storage: storage}
}

func (h *SnippetHandler) HandleSnippets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getSnippets(w, r)
	case http.MethodPost:
		h.createSnippet(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SnippetHandler) HandleSnippet(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/snippets/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getSnippet(w, r, id)
	case http.MethodPut:
		h.updateSnippet(w, r, id)
	case http.MethodDelete:
		h.deleteSnippet(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SnippetHandler) getSnippets(w http.ResponseWriter, r *http.Request) {
	snippets, err := h.storage.GetAll()
	if err != nil {
		http.Error(w, "Failed to retrieve snippets: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snippets)
}

func (h *SnippetHandler) getSnippet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	snippet, err := h.storage.Get(id)
	if err != nil {
		if err.Error() == "snippet not found" {
			http.Error(w, "Snippet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to retrieve snippet: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snippet)
}

func (h *SnippetHandler) createSnippet(w http.ResponseWriter, r *http.Request) {
	var snippet models.Snippet
	err := json.NewDecoder(r.Body).Decode(&snippet)
	if err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	snippet.ID = uuid.New()
	if snippet.Code == "" {
		http.Error(w, "Code cannot be empty", http.StatusBadRequest)
		return
	}
	err = h.storage.Create(snippet)
	if err != nil {
		http.Error(w, "Failed to create snippet: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(snippet)
}

func (h *SnippetHandler) updateSnippet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	var snippet models.Snippet
	err := json.NewDecoder(r.Body).Decode(&snippet)
	if err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if snippet.Code == "" {
		http.Error(w, "Code cannot be empty", http.StatusBadRequest)
		return
	}
	snippet.ID = id
	err = h.storage.Update(snippet)
	if err != nil {
		if err.Error() == "snippet not found" {
			http.Error(w, "Snippet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update snippet: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snippet)
}

func (h *SnippetHandler) deleteSnippet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	err := h.storage.Delete(id)
	if err != nil {
		if err.Error() == "snippet not found" {
			http.Error(w, "Snippet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete snippet: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// New handlers for tag and folder operations

func (h *SnippetHandler) HandleTags(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	snippetID, err := uuid.Parse(parts[2])
	if err != nil {
		http.Error(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}
	tagName := parts[3]

	switch r.Method {
	case http.MethodPost:
		h.addTag(w, r, snippetID, tagName)
	case http.MethodDelete:
		h.removeTag(w, r, snippetID, tagName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SnippetHandler) addTag(
	w http.ResponseWriter,
	r *http.Request,
	snippetID uuid.UUID,
	tagName string,
) {
	err := h.storage.AddTag(snippetID, tagName)
	if err != nil {
		http.Error(w, "Failed to add tag: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *SnippetHandler) removeTag(
	w http.ResponseWriter,
	r *http.Request,
	snippetID uuid.UUID,
	tagName string,
) {
	err := h.storage.RemoveTag(snippetID, tagName)
	if err != nil {
		http.Error(w, "Failed to remove tag: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SnippetHandler) HandleFolders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createFolder(w, r)
	case http.MethodGet:
		h.getFolderContents(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SnippetHandler) HandleUserFolders(w http.ResponseWriter, r *http.Request) {
	userIDStr := strings.TrimPrefix(r.URL.Path, "/folders/user/")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	folders, err := h.storage.GetFoldersByUser(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve folders: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func (h *SnippetHandler) createFolder(w http.ResponseWriter, r *http.Request) {
	var folder models.Folder
	err := json.NewDecoder(r.Body).Decode(&folder)
	if err != nil {
		http.Error(w, "Invalid request payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	folder.ID = uuid.New()
	err = h.storage.CreateFolder(folder)
	if err != nil {
		http.Error(w, "Failed to create folder: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func (h *SnippetHandler) getFolderContents(w http.ResponseWriter, r *http.Request) {
	folderIDStr := r.URL.Query().Get("id")
	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		http.Error(w, "Invalid folder ID", http.StatusBadRequest)
		return
	}

	snippets, folders, err := h.storage.GetFolderContents(folderID)
	if err != nil {
		http.Error(w, "Failed to get folder contents: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Snippets []models.Snippet `json:"snippets"`
		Folders  []models.Folder  `json:"folders"`
	}{
		Snippets: snippets,
		Folders:  folders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
