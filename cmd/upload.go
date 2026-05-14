package cmd

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
	"youtube-uploader/internal/youtube"
)

func RunUpload(videoPath, title, description, tagsStr, presetName string, cfg *config.Config) {
	var params youtube.UploadParams

	if presetName != "" {
		preset, err := db.GetPreset(presetName)
		if err != nil {
			log.Fatalf("Get preset: %v", err)
		}
		if preset == nil {
			log.Fatalf("Preset '%s' not found", presetName)
		}
		params.Tags = preset.Tags
		params.Description = preset.Description
		params.CategoryID = preset.CategoryID
		params.Privacy = preset.Privacy
		params.Title = preset.TitleTemplate
	}

	if title != "" {
		params.Title = title
	}
	if description != "" {
		params.Description = description
	}
	if tagsStr != "" {
		params.Tags = parseTags(tagsStr)
	}

	var videos []string
	if videoPath != "" {
		videos = []string{videoPath}
	} else {
		var err error
		videos, err = youtube.ListPendingVideos(cfg.VideosPending)
		if err != nil {
			log.Fatalf("List pending videos: %v", err)
		}
	}

	if len(videos) == 0 {
		fmt.Println("No videos to upload. Place .mp4 files in", cfg.VideosPending)
		return
	}

	accounts, err := db.ListAccounts()
	if err != nil {
		log.Fatalf("List accounts: %v", err)
	}
	if len(accounts) == 0 {
		fmt.Println("No accounts. Import accounts first.")
		return
	}

	for _, vp := range videos {
		p := params
		p.VideoPath = vp

		base := strings.TrimSuffix(filepath.Base(vp), filepath.Ext(vp))
		p.Title = strings.ReplaceAll(p.Title, "{filename}", base)

		if p.Title == "" {
			p.Title = base
		}

		fmt.Printf("\nUploading: %s\n  Title: %s\n  Accounts: %d\n", filepath.Base(vp), p.Title, len(accounts))

		results := youtube.UploadToAll(cfg, p, accounts, cfg.MaxConcurrency)

		success, failed := 0, 0
		for _, r := range results {
			if r.Error != nil {
				failed++
				log.Printf("  FAIL account %d: %v", r.AccountID, r.Error)
			} else {
				success++
				fmt.Printf("  OK   account %d: https://youtube.com/shorts/%s\n", r.AccountID, r.YoutubeID)
			}
		}

		fmt.Printf("  Results: %d success, %d failed\n", success, failed)

		if failed == 0 && success > 0 {
			if err := youtube.MoveToUploaded(vp, cfg.VideosUploaded); err != nil {
				log.Printf("Warning: could not move %s to uploaded: %v", vp, err)
			} else {
				fmt.Printf("  Moved to uploaded/\n")
			}
		}
	}
}

func parseTags(s string) []string {
	parts := strings.Split(s, ",")
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
