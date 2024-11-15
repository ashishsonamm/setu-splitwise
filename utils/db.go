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
	query := `
CREATE TABLE personal_settlements (
                                   id SERIAL PRIMARY KEY,
                                   debtor_id INT NOT NULL,
                                   creditor_id INT NOT NULL,
                                   amount FLOAT NOT NULL,
                                   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                   FOREIGN KEY (debtor_id) REFERENCES users(id) ON DELETE CASCADE,
                                   FOREIGN KEY (creditor_id) REFERENCES users(id) ON DELETE CASCADE
);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Table created successfully!")
}

func dropTables(db *sql.DB) {
	query := `
DROP TABLE IF EXISTS personal_settlements;
	DROP TABLE IF EXISTS group_settlements;
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
