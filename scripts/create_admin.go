// +build ignore

// Script to create an admin user for QuickQuote
// Run with: go run scripts/create_admin.go -email admin@example.com -password yourpassword
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	email := flag.String("email", "", "Admin email address")
	password := flag.String("password", "", "Admin password")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Println("Usage: go run scripts/create_admin.go -email <email> -password <password>")
		os.Exit(1)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://quickquote:quickquote@localhost:5432/quickquote?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Insert user
	_, err = pool.Exec(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET password_hash = $2
	`, *email, string(hash))
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("Admin user created/updated: %s\n", *email)
}
