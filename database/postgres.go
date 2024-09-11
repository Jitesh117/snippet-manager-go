package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

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
    CREATE TABLE IF NOT EXISTS users (
            id UUID PRIMARY KEY,
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
        );
    CREATE TABLE IF NOT EXISTS snippets (
        id UUID PRIMARY KEY,
        title TEXT NOT NULL,
        description TEXT,
        language TEXT NOT NULL,
        code TEXT NOT NULL,
        user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        folder_id UUID,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS tags (
        id UUID PRIMARY KEY,
        name TEXT NOT NULL UNIQUE
    );

    CREATE TABLE IF NOT EXISTS snippet_tags (
        snippet_id UUID REFERENCES snippets(id) ON DELETE CASCADE,
        tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
        PRIMARY KEY (snippet_id, tag_id)
    );

    CREATE TABLE IF NOT EXISTS folders (
        id UUID PRIMARY KEY,
        name TEXT NOT NULL,
        parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
        user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );
    `)
	return err
}

func (s *PostgresStorage) CreateUser(user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err = s.db.Exec(
		"INSERT INTO users (id, username, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		user.ID,
		user.Username,
		user.Email,
		hashedPassword,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (s *PostgresStorage) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, password, created_at, updated_at FROM users WHERE username = $1",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return user, err
}

func (s *PostgresStorage) GetUserByID(id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, username, email, created_at, updated_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return user, err
}

func (s *PostgresStorage) Create(snippet models.Snippet) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	snippet.CreatedAt = time.Now()
	snippet.UpdatedAt = time.Now()

	// First, insert the snippet without tags
	_, err = tx.Exec(
		"INSERT INTO snippets (id, title, description, language, code, user_id, folder_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		snippet.ID,
		snippet.Title,
		snippet.Description,
		snippet.Language,
		snippet.Code,
		snippet.UserID,
		snippet.FolderID,
		snippet.CreatedAt,
		snippet.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Then, add tags one by one
	for _, tag := range snippet.Tags {
		var tagID uuid.UUID
		err = tx.QueryRow("INSERT INTO tags (id, name) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id", uuid.New(), tag).
			Scan(&tagID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(
			"INSERT INTO snippet_tags (snippet_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			snippet.ID,
			tagID,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *PostgresStorage) Update(snippet models.Snippet) error {
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		return err
	}
	defer tx.Rollback()

	snippet.UpdatedAt = time.Now()
	_, err = tx.Exec(
		"UPDATE snippets SET title = $2, description = $3, language = $4, code = $5, folder_id = $6, updated_at = $7 WHERE id = $1",
		snippet.ID,
		snippet.Title,
		snippet.Description,
		snippet.Language,
		snippet.Code,
		snippet.FolderID,
		snippet.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error updating snippet: %v", err)
		return err
	}

	log.Println("Snippet updated successfully")

	// Remove existing tags
	_, err = tx.Exec("DELETE FROM snippet_tags WHERE snippet_id = $1", snippet.ID)
	if err != nil {
		log.Printf("Error deleting snippet tags: %v", err)
		return err
	}

	log.Println("Snippet tags deleted successfully")

	// Add new tags
	for _, tag := range snippet.Tags {
		var tagID uuid.UUID
		err := tx.QueryRow("SELECT id FROM tags WHERE name = $1", tag).Scan(&tagID)
		if err != nil {
			if err == sql.ErrNoRows {
				// Tag does not exist, create it
				tagID = uuid.New()
				_, err := tx.Exec("INSERT INTO tags (id, name) VALUES ($1, $2)", tagID, tag)
				if err != nil {
					log.Printf("Error creating tag %s: %v", tag, err)
					return err
				}
			} else {
				log.Printf("Error querying tag %s: %v", tag, err)
				return err
			}
		}

		_, err = tx.Exec(
			"INSERT INTO snippet_tags (snippet_id, tag_id) VALUES ($1, $2)",
			snippet.ID,
			tagID,
		)
		if err != nil {
			log.Printf("Error adding tag %s to snippet %v: %v", tag, snippet.ID, err)
			return err
		}
		log.Printf("Tag %s added successfully", tag)
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
		return err
	}

	log.Println("Transaction committed successfully")
	return nil
}

func (s *PostgresStorage) GetAll() ([]models.Snippet, error) {
	rows, err := s.db.Query(
		"SELECT id, title, description, language, code, user_id, folder_id, created_at, updated_at FROM snippets",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snippets []models.Snippet
	for rows.Next() {
		var snip models.Snippet
		if err := rows.Scan(&snip.ID, &snip.Title, &snip.Description, &snip.Language, &snip.Code, &snip.UserID, &snip.FolderID, &snip.CreatedAt, &snip.UpdatedAt); err != nil {
			return nil, err
		}
		tags, err := s.GetSnippetTags(snip.ID)
		if err != nil {
			return nil, err
		}
		snip.Tags = tags
		snippets = append(snippets, snip)
	}
	return snippets, nil
}

func (s *PostgresStorage) Get(id uuid.UUID) (models.Snippet, error) {
	var snip models.Snippet
	err := s.db.QueryRow("SELECT id, title, description, language, code, user_id, folder_id, created_at, updated_at FROM snippets WHERE id = $1", id).
		Scan(&snip.ID, &snip.Title, &snip.Description, &snip.Language, &snip.Code, &snip.UserID, &snip.FolderID, &snip.CreatedAt, &snip.UpdatedAt)
	if err != nil {
		return snip, fmt.Errorf("snippet not found")
	}
	tags, err := s.GetSnippetTags(id)
	if err != nil {
		return snip, err
	}
	snip.Tags = tags
	return snip, nil
}

func (s *PostgresStorage) Delete(id uuid.UUID) error {
	_, err := s.db.Exec("DELETE FROM snippets where id = $1", id)
	return err
}

func (s *PostgresStorage) AddTag(snippetID uuid.UUID, tagName string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tagID uuid.UUID
	err = tx.QueryRow("INSERT INTO tags (id, name) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id", uuid.New(), tagName).
		Scan(&tagID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		"INSERT INTO snippet_tags (snippet_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		snippetID,
		tagID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *PostgresStorage) RemoveTag(snippetID uuid.UUID, tagName string) error {
	_, err := s.db.Exec(`
        DELETE FROM snippet_tags
        WHERE snippet_id = $1 AND tag_id = (SELECT id FROM tags WHERE name = $2)
    `, snippetID, tagName)
	return err
}

func (s *PostgresStorage) GetSnippetTags(snippetID uuid.UUID) ([]string, error) {
	rows, err := s.db.Query(`
        SELECT t.name
        FROM tags t
        JOIN snippet_tags st ON t.id = st.tag_id
        WHERE st.snippet_id = $1
    `, snippetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (s *PostgresStorage) CreateFolder(folder models.Folder) error {
	folder.CreatedAt = time.Now()
	folder.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO folders (id, name, parent_id, user_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		folder.ID,
		folder.Name,
		folder.ParentID,
		folder.UserID,
		folder.CreatedAt,
		folder.UpdatedAt,
	)
	return err
}

func (s *PostgresStorage) GetFoldersByUser(userID uuid.UUID) ([]models.Folder, error) {
	rows, err := s.db.Query(
		"SELECT id, name, user_id, parent_id, created_at, updated_at FROM folders WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []models.Folder
	for rows.Next() {
		var folder models.Folder
		if err := rows.Scan(&folder.ID, &folder.Name, &folder.UserID, &folder.ParentID, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return folders, nil
}

func (s *PostgresStorage) GetFolderContents(
	folderID uuid.UUID,
) ([]models.Snippet, []models.Folder, error) {
	snippets, err := s.db.Query(
		"SELECT id, title, description, language, code, user_id, folder_id, created_at, updated_at FROM snippets WHERE folder_id = $1",
		folderID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer snippets.Close()

	var snippetList []models.Snippet
	for snippets.Next() {
		var snip models.Snippet
		if err := snippets.Scan(&snip.ID, &snip.Title, &snip.Description, &snip.Language, &snip.Code, &snip.UserID, &snip.FolderID, &snip.CreatedAt, &snip.UpdatedAt); err != nil {
			return nil, nil, err
		}
		tags, err := s.GetSnippetTags(snip.ID)
		if err != nil {
			return nil, nil, err
		}
		snip.Tags = tags
		snippetList = append(snippetList, snip)
	}

	folders, err := s.db.Query(
		"SELECT id, name, parent_id, user_id, created_at, updated_at FROM folders WHERE parent_id = $1",
		folderID,
	)
	if err != nil {
		return nil, nil, err
	}
	defer folders.Close()

	var folderList []models.Folder
	for folders.Next() {
		var folder models.Folder
		if err := folders.Scan(&folder.ID, &folder.Name, &folder.ParentID, &folder.UserID, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, nil, err
		}
		folderList = append(folderList, folder)
	}

	return snippetList, folderList, nil
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}
