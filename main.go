package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"youtube-uploader/cmd"
	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
)

const usage = `YouTube Shorts Uploader

Usage:
  youtube-uploader <command> [flags]

Commands:
  import          Import accounts from file (client_id:client_secret)
  auth            Authorize accounts via OAuth2
  account         Manage accounts (list/delete/restore)
  upload          Upload videos to all authorized accounts
  preset          Manage video metadata presets
  schedule        Manage upload schedules
  daemon          Start scheduler daemon
  serve           Start web panel

Run 'youtube-uploader <command> --help' for details.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(0)
	}

	cfg := config.Load()

	if err := db.Open(cfg.DBPath); err != nil {
		log.Fatalf("Database: %v", err)
	}
	defer db.Close()
	if err := db.RunMigrations("migrations"); err != nil {
		log.Fatalf("Migrations: %v", err)
	}

	command := os.Args[1]
	os.Args = os.Args[1:]

	switch command {
	case "import":
		fs := flag.NewFlagSet("import", flag.ExitOnError)
		file := fs.String("file", "accounts.txt", "Path to accounts file")
		fs.Parse(os.Args[1:])
		cmd.RunImport(*file)

	case "auth":
		fs := flag.NewFlagSet("auth", flag.ExitOnError)
		account := fs.String("account", "all", "Account ID or 'all'")
		fs.Parse(os.Args[1:])
		cmd.RunAuth(*account, cfg)

	case "account":
		if len(os.Args) < 2 {
			fmt.Println("Usage: account <list|delete|restore> [id]")
			return
		}
		sub := os.Args[1]
		switch sub {
		case "list":
			cmd.RunAccountList()
		case "delete":
			if len(os.Args) < 3 {
				log.Fatal("Usage: account delete <id>")
			}
			cmd.RunAccountDelete(os.Args[2])
		case "restore":
			if len(os.Args) < 3 {
				log.Fatal("Usage: account restore <id>")
			}
			cmd.RunAccountRestore(os.Args[2])
		default:
			fmt.Printf("Unknown account subcommand: %s\n", sub)
		}

	case "upload":
		fs := flag.NewFlagSet("upload", flag.ExitOnError)
		video := fs.String("video", "", "Path to video file (or uploads all from pending/)")
		title := fs.String("title", "", "Video title")
		desc := fs.String("description", "", "Video description")
		tags := fs.String("tags", "", "Comma-separated tags")
		preset := fs.String("preset", "", "Use preset name")
		fs.Parse(os.Args[1:])
		cmd.RunUpload(*video, *title, *desc, *tags, *preset, cfg)

	case "preset":
		if len(os.Args) < 2 {
			fmt.Println("Usage: preset <create|list>")
			return
		}
		sub := os.Args[1]
		switch sub {
		case "create":
			fs := flag.NewFlagSet("preset create", flag.ExitOnError)
			name := fs.String("name", "", "Preset name")
			title := fs.String("title", "{filename}", "Title template")
			desc := fs.String("description", "", "Description")
			tags := fs.String("tags", "", "Comma-separated tags")
			cat := fs.String("category", "22", "YouTube category ID")
			privacy := fs.String("privacy", "public", "Privacy: public/unlisted/private")
			fs.Parse(os.Args[2:])
			cmd.RunPresetCreate(*name, *title, *desc, *tags, *cat, *privacy)
		case "list":
			cmd.RunPresetList()
		default:
			fmt.Printf("Unknown preset subcommand: %s\n", sub)
		}

	case "schedule":
		if len(os.Args) < 2 {
			fmt.Println("Usage: schedule <add|list|enable|disable>")
			return
		}
		sub := os.Args[1]
		switch sub {
		case "add":
			fs := flag.NewFlagSet("schedule add", flag.ExitOnError)
			cron := fs.String("cron", "", "Cron expression (5 fields)")
			title := fs.String("title", "", "Video title")
			desc := fs.String("description", "", "Description")
			tags := fs.String("tags", "", "Comma-separated tags")
			preset := fs.String("preset", "", "Preset name")
			fs.Parse(os.Args[2:])
			cmd.RunScheduleAdd(*cron, *title, *desc, *tags, *preset)
		case "list":
			cmd.RunScheduleList()
		case "enable":
			if len(os.Args) < 3 {
				log.Fatal("Usage: schedule enable <id>")
			}
			cmd.RunScheduleToggle(os.Args[2], true)
		case "disable":
			if len(os.Args) < 3 {
				log.Fatal("Usage: schedule disable <id>")
			}
			cmd.RunScheduleToggle(os.Args[2], false)
		default:
			fmt.Printf("Unknown schedule subcommand: %s\n", sub)
		}

	case "daemon":
		cmd.RunDaemon(cfg)

	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		port := fs.String("port", "", "Override port")
		fs.Parse(os.Args[1:])
		if *port != "" {
			cfg.Port = *port
		}
		cmd.RunServe(cfg)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}
