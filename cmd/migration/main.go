package main

import (
	"errors"
	"log"
	"os"
	"strconv"

	"github.com/ankur59/classifieds-app/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
)

// main runs database migrations from ./migrations.
// Usage: migration up | migration down <steps>
func main() {
	if len(os.Args) < 2 {
		log.Fatal("Missing args with migration command make sure to pass up or down as argument")
	}
	cfg := config.MustLoad()
	c, err := migrate.New("file://./migrations", cfg.DB_URL)
	if err != nil {
		log.Fatalf("migration.new %v", err)
	}

	switch os.Args[1] {
	// Apply all pending migrations; "no change" is not an error.
	case "up":
		if err := c.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("migration.up %v", err)
		}
	// Roll back the given number of migrations.
	case "down":
		if len(os.Args) < 3 {
			log.Fatalf("Down migration steps not provided")
		}
		steps, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("migration.down.stringconvr %v", err)
		}

		if err := c.Steps(-steps); err != nil {
			log.Fatalf("migration.down %v", err)
		}
	default:
		log.Fatalf("unknown migration command %q, expected up or down", os.Args[1])
	}
}
