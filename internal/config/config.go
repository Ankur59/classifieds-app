package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT   string
	DB_URL string
}

// Must load function don't return error if error occurs code execution should stop there
func MustLoad() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load env")
	}
	PORT := os.Getenv("PORT")
	if PORT == "" {
		log.Fatal("Failed to load PORT from env")
	}

	DB_URL := os.Getenv("DB_URL")
	if DB_URL == "" {
		log.Fatal("Failed to load DB_URL from env")
	}

	return Config{
		PORT:   PORT,
		DB_URL: DB_URL,
	}
}
