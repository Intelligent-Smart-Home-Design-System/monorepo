package main

import (
    "database/sql"
    "fmt"
    "os"
    _ "github.com/lib/pq"
)

func main() {
    connStr := "host=localhost user=scraper password=1234 dbname=scraper_db sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        panic(err)
    }
    defer db.Close()
    var archive []byte
    err = db.QueryRow("SELECT warc_bundle_archive FROM page_snapshots ORDER BY id DESC LIMIT 1").Scan(&archive)
    if err != nil {
        panic(err)
    }
    err = os.WriteFile("/tmp/test.tar.gz", archive, 0644)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Saved %d bytes to /tmp/test.tar.gz\n", len(archive))
}