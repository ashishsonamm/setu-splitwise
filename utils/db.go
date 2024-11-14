package utils

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	connStr := os.Getenv("DATABASE_URL")
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatalf("Database is not reachable: %v", err)
	}

	log.Println("Connected to the database successfully!")

	//createTables(DB)
	//dropTables(DB)
}

func createTables(db *sql.DB) {
	query := `CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       name VARCHAR(100) NOT NULL,
                       email VARCHAR(100) UNIQUE NOT NULL,
                       password VARCHAR(100),
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE groups (
                        id SERIAL PRIMARY KEY,
                        name VARCHAR(100) NOT NULL,
                        created_by INT REFERENCES users(id) ON DELETE SET NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE group_users (
                             id SERIAL PRIMARY KEY,
                             group_id INT REFERENCES groups(id) ON DELETE CASCADE,
                             user_id INT REFERENCES users(id) ON DELETE CASCADE,
                             joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                             UNIQUE (group_id, user_id)
);


CREATE TABLE expenses (
                          id SERIAL PRIMARY KEY,
                          description TEXT NOT NULL,
                          amount FLOAT NOT NULL,
                          split_type VARCHAR(20) NOT NULL,
                          expense_type VARCHAR(20) NOT NULL,
                          created_by INT NOT NULL,
                          group_id INT
);

CREATE TABLE contributors (
                              id SERIAL PRIMARY KEY,
                              expense_id INT REFERENCES expenses(id) ON DELETE CASCADE,
                              user_id INT NOT NULL,
                              contribution_amount FLOAT NOT NULL,
                              paid_amount FLOAT,
                              percentage FLOAT,
                              share FLOAT,
                              amount FLOAT
);

CREATE TABLE amounts_owed (
                              id SERIAL PRIMARY KEY,
                              expense_id INT REFERENCES expenses(id) ON DELETE CASCADE,
                              user_id INT NOT NULL,
                              owed FLOAT DEFAULT 0,
                              balance FLOAT DEFAULT 0
);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Table created successfully!")
}

func dropTables(db *sql.DB) {
	query := `
   DROP TABLE IF EXISTS contributors;
   DROP TABLE IF EXISTS amounts_owed;
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS group_users;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Table dropped successfully!")
}
