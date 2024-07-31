package databse

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"snippet-manager-go/models"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(host, port, user, password, dbname string) (*PostgresStorage, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Init() error {
	_, err := s.db.Exec(`
    CREATE TABLE IF NOT EXISTS snippets (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    language TEXT NOT NULL,
    code TEXT NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    )
   `)
	return err
}

func (s *PostgresStorage) Create(snippet models.Snippet) error {
	snippet.CreatedAt = time.Now()
	snippet.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO snippets (id, title, description, language, code, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		snippet.ID,
		snippet.Title,
		snippet.Description,
		snippet.Language,
		snippet.Code,
		snippet.UserID,
		snippet.CreatedAt,
		snippet.UpdatedAt,
	)
	return err
}

func (s *PostgresStorage) Update(snippet models.Snippet) error {
	snippet.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE snippets SET title = $2, description = $3, language = $4, code = $5, updated_at = $6 WHERE id = $1",
		snippet.ID,
		snippet.Title,
		snippet.Description,
		snippet.Language,
		snippet.Code,
		snippet.UpdatedAt,
	)
	return err
}

func (s *PostgresStorage) GetAll() ([]models.Snippet, error) {
	rows, err := s.db.Query(
		"SELECT id, title, description, language, code, user_id, created_at, updated_at FROM snippets",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var snippets []models.Snippet
	for rows.Next() {
		var snip models.Snippet
		if err := rows.Scan(&snip.ID, &snip.Title, &snip.Description, &snip.Language, &snip.Code, &snip.UserID, &snip.CreatedAt, &snip.UpdatedAt); err != nil {
			return nil, err
		}
		snippets = append(snippets, snip)
	}
	return snippets, nil
}

func (s *PostgresStorage) Get(id uuid.UUID) (models.Snippet, error) {
	var snip models.Snippet
	err := s.db.QueryRow("SELECT id, title, description, language, code, user_id, created_at, updated_at FROM snippets WHERE id = $1", id).
		Scan(&snip.ID, &snip.Description, &snip.Language, &snip.Code, &snip.UserID, &snip.CreatedAt, &snip.UpdatedAt)
	if err != nil {
		return snip, fmt.Errorf("snippet not found")
	}
	return snip, err
}

func (s *PostgresStorage) Delete(id uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM snippets where id = $1", id)
	return err
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}
