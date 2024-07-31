package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	database "snippet-manager-go/database"
	"snippet-manager-go/models"
)

type SnippetHandler struct {
	storage *database.PostgresStorage
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
	idStr := strings.TrimPrefix(r.URL.Path, "/snippets")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid snippet ID", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.GetSnippet(w, r, id)
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
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
