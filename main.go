package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/joho/godotenv"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func loadConfig() *Config {
	cfg := &Config{
		Host:     getEnv("DB_HOST", ""),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", ""),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", ""),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" || cfg.DBName == "" {
		log.Fatal("Missing required database configuration: DB_HOST, DB_USER, DB_PASSWORD, DB_NAME")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: loading .env: %v", err)
	}

	var (
		command        = flag.String("command", "up", "Migration command: up, down, force, version")
		steps          = flag.Int("steps", 0, "Number of migration steps (for up/down commands, 0 = all)")
		version        = flag.Int("version", 0, "Version to force (for force command)")
		schema         = flag.String("schema", "", "Database schema name (required)")
		migrationsPath = flag.String("path", "", "Path to migrations directory (required)")
	)
	flag.Parse()

	if *schema == "" {
		log.Fatal("Schema name is required: use -schema flag")
	}

	if *migrationsPath == "" {
		log.Fatal("Migrations path is required: use -path flag")
	}

	cfg := loadConfig()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	if err := createSchemaIfNotExists(db, *schema); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	absPath, err := filepath.Abs(*migrationsPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Fatalf("Migrations directory not found: %s", absPath)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "schema_migrations",
		SchemaName:      *schema,
	})
	if err != nil {
		log.Fatalf("Failed to create postgres driver: %v", err)
	}

	sourceURL := fmt.Sprintf("file://%s", absPath)
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	switch *command {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration failed: %v", err)
		}
		if err == migrate.ErrNoChange {
			log.Println("No migrations to apply")
		} else {
			log.Println("Migrations applied successfully")
		}

	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
		if err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration failed: %v", err)
		}
		if err == migrate.ErrNoChange {
			log.Println("No migrations to rollback")
		} else {
			log.Println("Migrations rolled back successfully")
		}

	case "force":
		if *version == 0 {
			log.Fatal("Version is required for force command")
		}
		if err := m.Force(*version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		log.Printf("Version forced to: %d", *version)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if err == migrate.ErrNilVersion {
				fmt.Println("Version: (no migrations applied)")
				return
			}
			log.Fatalf("Failed to get version: %v", err)
		}
		if dirty {
			fmt.Printf("Version: %d (dirty)\n", version)
		} else {
			fmt.Printf("Version: %d\n", version)
		}

	default:
		log.Fatalf("Unknown command: %s. Use: up, down, force, version", *command)
	}
}

func createSchemaIfNotExists(db *sql.DB, schemaName string) error {
	var exists bool
	checkSQL := `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`
	err := db.QueryRow(checkSQL, schemaName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check schema: %w", err)
	}

	if !exists {
		createSQL := fmt.Sprintf("CREATE SCHEMA %s", schemaName)
		_, err = db.Exec(createSQL)
		if err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}
		log.Printf("Schema '%s' created", schemaName)
	}

	return nil
}
