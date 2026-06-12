package repository

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "testing"
    "time"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func setupTestDatabase(t *testing.T) (*sql.DB, func()) {
    t.Helper()

    host := getEnv("TEST_PG_HOST", "localhost")
    port := getEnv("TEST_PG_PORT", "5432")
    user := getEnv("TEST_PG_USER", "postgres")
    password := getEnv("TEST_PG_PASSWORD", "")
    dbname := getEnv("TEST_PG_DBNAME", "postgres")
    sslmode := getEnv("TEST_PG_SSLMODE", "disable")

    adminConnStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        host, port, user, password, dbname, sslmode)
    adminDB, err := sql.Open("postgres", adminConnStr)
    if err != nil {
        t.Skipf("Skipping integration test: cannot connect to admin DB: %v", err)
        return nil, func() {}
    }
    defer adminDB.Close()

    testDBName := fmt.Sprintf("test_db_%d", time.Now().UnixNano())
    _, err = adminDB.Exec("CREATE DATABASE " + testDBName)
    if err != nil {
        t.Skipf("Skipping integration test: cannot create test DB: %v", err)
        return nil, func() {}
    }

    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        host, port, user, password, testDBName, sslmode)
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        t.Skipf("Skipping integration test: cannot connect to test DB: %v", err)
        return nil, func() {}
    }

    migrateURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
        user, password, host, port, testDBName, sslmode)

    _, currentFile, _, _ := runtime.Caller(0)
    baseDir := filepath.Dir(currentFile)
    migrationsPath := "file://" + filepath.Join(baseDir, "../../../db/catalog/migrations")
    migrateInstance, err := migrate.New(migrationsPath, migrateURL)
    if err != nil {
        t.Skipf("Skipping integration test: cannot create migrate instance: %v", err)
        return nil, func() {}
    }
    if err := migrateInstance.Up(); err != nil && err != migrate.ErrNoChange {
        t.Skipf("Skipping integration test: failed to apply migrations: %v", err)
        return nil, func() {}
    }

    cleanup := func() {
        db.Close()
        _, _ = adminDB.Exec("DROP DATABASE " + testDBName)
    }

    return db, cleanup
}