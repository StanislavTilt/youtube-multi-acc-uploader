package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
	"youtube-uploader/internal/scheduler"
	"youtube-uploader/internal/youtube"
)

func RunDaemon(cfg *config.Config) {
	fmt.Println("Daemon started. Checking schedules every 60 seconds.")
	fmt.Println("Press Ctrl+C to stop.")

	initScheduleNextRun()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	runDueSchedules(cfg)

	for {
		select {
		case <-ticker.C:
			runDueSchedules(cfg)
		case <-sigCh:
			fmt.Println("\nDaemon stopped.")
			return
		}
	}
}

func initScheduleNextRun() {
	schedules, err := db.ListSchedules()
	if err != nil {
		log.Printf("Warning: list schedules: %v", err)
		return
	}
	for _, s := range schedules {
		if s.Enabled && s.NextRunAt == nil {
			next, err := scheduler.NextCronRun(s.CronExpr, time.Now())
			if err != nil {
				log.Printf("Warning: invalid cron for schedule #%d: %v", s.ID, err)
				continue
			}
			db.UpdateScheduleLastRun(s.ID, &next)
		}
	}
}

func runDueSchedules(cfg *config.Config) {
	schedules, err := db.GetDueSchedules()
	if err != nil {
		log.Printf("Error getting due schedules: %v", err)
		return
	}

	for _, s := range schedules {
		log.Printf("Running schedule #%d (%s)", s.ID, s.CronExpr)

		var videoPath string
		if s.VideoPath != nil && *s.VideoPath != "" {
			videoPath = *s.VideoPath
		} else {

			videos, err := youtube.ListPendingVideos(cfg.VideosPending)
			if err != nil || len(videos) == 0 {
				log.Printf("Schedule #%d: no pending videos", s.ID)
				updateNextRun(s)
				continue
			}
			videoPath = videos[0]
		}

		title, desc, tags := s.Title, s.Description, s.Tags
		categoryID, privacy := "22", "public"
		if s.PresetID != nil {

			presets, _ := db.ListPresets()
			for _, p := range presets {
				if p.ID == *s.PresetID {
					if title == "" {
						title = p.TitleTemplate
					}
					if desc == "" {
						desc = p.Description
					}
					if len(tags) == 0 {
						tags = p.Tags
					}
					categoryID = p.CategoryID
					privacy = p.Privacy
					break
				}
			}
		}

		base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
		title = strings.ReplaceAll(title, "{filename}", base)
		title = strings.ReplaceAll(title, "{date}", time.Now().Format("2006-01-02"))
		if title == "" {
			title = base
		}

		params := youtube.UploadParams{
			VideoPath:	videoPath,
			Title:		title,
			Description:	desc,
			Tags:		tags,
			CategoryID:	categoryID,
			Privacy:	privacy,
		}

		accounts, err := db.ListAccounts()
		if err != nil {
			log.Printf("Schedule #%d: list accounts: %v", s.ID, err)
			updateNextRun(s)
			continue
		}

		if len(s.AccountIDs) > 0 {
			idSet := make(map[int64]bool)
			for _, id := range s.AccountIDs {
				idSet[id] = true
			}
			var filtered []db.Account
			for _, a := range accounts {
				if idSet[a.ID] {
					filtered = append(filtered, a)
				}
			}
			accounts = filtered
		}

		results := youtube.UploadToAll(cfg, params, accounts, cfg.MaxConcurrency)

		success, failed := 0, 0
		for _, r := range results {
			if r.Error != nil {
				failed++
			} else {
				success++
			}
		}
		log.Printf("Schedule #%d done: %d success, %d failed", s.ID, success, failed)

		if failed == 0 && success > 0 {
			youtube.MoveToUploaded(videoPath, cfg.VideosUploaded)
		}

		updateNextRun(s)
	}
}

func updateNextRun(s db.Schedule) {
	next, err := scheduler.NextCronRun(s.CronExpr, time.Now())
	if err != nil {
		log.Printf("Warning: next run for schedule #%d: %v", s.ID, err)
		return
	}
	db.UpdateScheduleLastRun(s.ID, &next)
	log.Printf("Schedule #%d next run: %s", s.ID, next.Format("2006-01-02 15:04"))
}
