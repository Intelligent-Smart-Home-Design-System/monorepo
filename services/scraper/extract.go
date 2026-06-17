package main

import (
    "database/sql"
    "os"

    "github.com/rs/zerolog"
    _ "github.com/lib/pq"
)

func main() {
    log := zerolog.New(os.Stderr).With().Timestamp().Logger()

    connStr := "host=localhost user=scraper password=1234 dbname=scraper_db sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to open database")
    }
    defer db.Close()
    var archive []byte
    err = db.QueryRow("SELECT warc_bundle_archive FROM page_snapshots ORDER BY id DESC LIMIT 1").Scan(&archive)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to query archive")
    }
    err = os.WriteFile("/tmp/test.tar.gz", archive, 0644)
    if err != nil {
        log.Fatal().Err(err).Msg("failed to write archive file")
    }
    log.Debug().Int("bytes", len(archive)).Msg("Bytes saved to /tmp/test.tar.gz")
}