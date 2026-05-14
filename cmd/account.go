package cmd

import (
	"fmt"
	"log"
	"strconv"

	"youtube-uploader/internal/db"
)

func RunAccountList() {
	accounts, err := db.ListAllAccounts()
	if err != nil {
		log.Fatalf("List accounts: %v", err)
	}
	if len(accounts) == 0 {
		fmt.Println("No accounts. Import with: import --file accounts.txt")
		return
	}
	fmt.Printf("%-4s %-40s %-10s %s\n", "ID", "CLIENT ID", "STATUS", "NAME")
	for _, a := range accounts {
		status := "active"
		if a.DeletedAt != nil {
			status = "deleted"
		}
		cid := a.ClientID
		if len(cid) > 38 {
			cid = cid[:35] + "..."
		}
		tok, _ := db.GetToken(a.ID)
		auth := ""
		if tok != nil && tok.ChannelName != "" {
			auth = fmt.Sprintf(" [%s]", tok.ChannelName)
		} else if tok != nil {
			auth = " [authorized]"
		}
		fmt.Printf("%-4d %-40s %-10s %s%s\n", a.ID, cid, status, a.Name, auth)
	}
}

func RunAccountDelete(idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid account ID: %s", idStr)
	}
	if err := db.SoftDeleteAccount(id); err != nil {
		log.Fatalf("Delete account: %v", err)
	}
	fmt.Printf("Account #%d soft-deleted\n", id)
}

func RunAccountRestore(idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid account ID: %s", idStr)
	}
	if err := db.RestoreAccount(id); err != nil {
		log.Fatalf("Restore account: %v", err)
	}
	fmt.Printf("Account #%d restored\n", id)
}
