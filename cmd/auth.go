package cmd

import (
	"fmt"
	"log"
	"strconv"

	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
	"youtube-uploader/internal/youtube"
)

func RunAuth(accountArg string, cfg *config.Config) {
	if accountArg == "all" {
		accounts, err := db.ListAccounts()
		if err != nil {
			log.Fatalf("List accounts: %v", err)
		}
		if len(accounts) == 0 {
			fmt.Println("No accounts found. Import or add accounts first.")
			return
		}
		for i := range accounts {
			if err := youtube.AuthorizeCLI(cfg, &accounts[i]); err != nil {
				log.Printf("Auth failed for account %d: %v", accounts[i].ID, err)
			} else {
				fmt.Printf("Account %d authorized successfully\n", accounts[i].ID)
			}
		}
		return
	}

	id, err := strconv.ParseInt(accountArg, 10, 64)
	if err != nil {
		log.Fatalf("Invalid account ID: %s", accountArg)
	}
	acc, err := db.GetAccount(id)
	if err != nil {
		log.Fatalf("Get account: %v", err)
	}
	if acc == nil {
		log.Fatalf("Account %d not found", id)
	}
	if err := youtube.AuthorizeCLI(cfg, acc); err != nil {
		log.Fatalf("Auth failed: %v", err)
	}
	fmt.Printf("Account %d authorized successfully\n", id)
}
