package main

import (
	"fmt"
	"log"
	"net/http"

	database "snippet-manager-go/database"
	"snippet-manager-go/handlers"
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

	http.HandleFunc("/snippets/", snippetHandler.HandleSnippet)
	http.HandleFunc("/snippets", snippetHandler.HandleSnippets)
	http.HandleFunc("/tags/", snippetHandler.HandleTags)

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
