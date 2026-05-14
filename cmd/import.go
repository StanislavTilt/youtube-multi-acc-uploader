package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"youtube-uploader/internal/db"
)

func RunImport(filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Cannot open file: %v", err)
	}
	defer f.Close()

	var imported, skipped int
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		clientID, clientSecret, ok := strings.Cut(line, ":")
		if !ok {
			log.Printf("Skipping invalid line: %s", line)
			skipped++
			continue
		}
		clientID = strings.TrimSpace(clientID)
		clientSecret = strings.TrimSpace(clientSecret)

		id, err := db.InsertAccount(clientID, clientSecret)
		if err != nil {
			log.Printf("Error inserting: %v", err)
			skipped++
			continue
		}
		if id == 0 {
			skipped++
		} else {
			imported++
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Read error: %v", err)
	}

	fmt.Printf("Import complete: %d added, %d skipped\n", imported, skipped)
}
