package main

import (
	"fmt"
	"log"
	"net/http"

	database "snippet-manager-go/database"
	"snippet-manager-go/handlers"
	"snippet-manager-go/middleware"
)

func main() {
	store, err := database.NewPostgresStorage(
		"localhost",
		"5432",
		"postgres",
		"mysecretpassword",
		"snippet_manager",
	)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	snippetHandler := handlers.NewSnippetHandler(store)
	userHandler := handlers.NewUserHandler(store)

	// Public routes
	http.HandleFunc("/register", userHandler.Register)
	http.HandleFunc("/login", userHandler.Login)

	// Protected routes
	http.HandleFunc("/snippets/", middleware.JWTAuth(snippetHandler.HandleSnippet))
	http.HandleFunc("/snippets", middleware.JWTAuth(snippetHandler.HandleSnippets))
	http.HandleFunc("/tags/", middleware.JWTAuth(snippetHandler.HandleTags))
	http.HandleFunc("/folders", middleware.JWTAuth(snippetHandler.HandleFolders))
	http.HandleFunc("/folders/user/", middleware.JWTAuth(snippetHandler.HandleUserFolders))

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
