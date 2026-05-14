package youtube

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/option"
	ytapi "google.golang.org/api/youtube/v3"

	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
	"youtube-uploader/internal/uniquify"
)

type UploadParams struct {
	VideoPath	string
	Title		string
	Description	string
	Tags		[]string
	CategoryID	string
	Privacy		string
}

type UploadResult struct {
	AccountID	int64
	UploadID	int64
	YoutubeID	string
	Error		error
}

func UploadToAccount(client *http.Client, params UploadParams, accountID, uploadID int64) UploadResult {
	result := UploadResult{AccountID: accountID, UploadID: uploadID}
	db.UpdateUploadStatus(uploadID, "uploading", "", "")

	ctx := context.Background()
	svc, err := ytapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		result.Error = fmt.Errorf("create youtube service: %w", err)
		db.UpdateUploadStatus(uploadID, "failed", "", result.Error.Error())
		return result
	}

	file, err := os.Open(params.VideoPath)
	if err != nil {
		result.Error = fmt.Errorf("open video: %w", err)
		db.UpdateUploadStatus(uploadID, "failed", "", result.Error.Error())
		return result
	}
	defer file.Close()

	privacy := params.Privacy
	if privacy == "" {
		privacy = "public"
	}
	categoryID := params.CategoryID
	if categoryID == "" {
		categoryID = "22"
	}

	desc := params.Description
	if len(params.Tags) > 0 {
		hashtags := "\n\n"
		for _, t := range params.Tags {
			cleaned := strings.ReplaceAll(strings.TrimSpace(t), " ", "")
			if cleaned != "" {
				if !strings.HasPrefix(cleaned, "#") {
					cleaned = "#" + cleaned
				}
				hashtags += cleaned + " "
			}
		}
		desc += hashtags
	}
	if !strings.Contains(strings.ToLower(desc), "#shorts") {
		desc += "#Shorts "
	}

	sanitizedTitle := strings.ReplaceAll(params.Title, "<3", "❤️")
	sanitizedTitle = strings.NewReplacer("<", "", ">", "").Replace(sanitizedTitle)
	if len(sanitizedTitle) > 100 {
		sanitizedTitle = sanitizedTitle[:100]
	}

	video := &ytapi.Video{
		Snippet: &ytapi.VideoSnippet{
			Title:		sanitizedTitle,
			Description:	strings.TrimSpace(desc),
			Tags:		params.Tags,
			CategoryId:	categoryID,
		},
		Status: &ytapi.VideoStatus{
			PrivacyStatus:	privacy,
			MadeForKids:	false,
		},
	}

	call := svc.Videos.Insert([]string{"snippet", "status"}, video)
	call.Media(file)

	resp, err := call.Do()
	if err != nil {
		result.Error = fmt.Errorf("upload video: %w", err)
		db.UpdateUploadStatus(uploadID, "failed", "", result.Error.Error())
		return result
	}

	result.YoutubeID = resp.Id
	db.UpdateUploadStatus(uploadID, "done", resp.Id, "")
	log.Printf("Uploaded to account %d: https://youtube.com/shorts/%s", accountID, resp.Id)
	return result
}

func UploadToAll(cfg *config.Config, params UploadParams, accounts []db.Account, maxConcurrency int) []UploadResult {
	sem := make(chan struct{}, maxConcurrency)
	var mu sync.Mutex
	var results []UploadResult
	var wg sync.WaitGroup

	for _, acc := range accounts {
		tok, err := db.GetToken(acc.ID)
		if err != nil || tok == nil {
			log.Printf("Skipping account %d: no token", acc.ID)
			continue
		}

		client, err := GetOAuth2Client(cfg, &acc, tok)
		if err != nil {
			log.Printf("Skipping account %d: %v", acc.ID, err)
			continue
		}

		uploadID, err := db.InsertUpload(acc.ID, params.VideoPath, params.Title, params.Description, params.Tags)
		if err != nil {
			log.Printf("Skipping account %d: insert upload: %v", acc.ID, err)
			continue
		}

		wg.Add(1)
		go func(c *http.Client, aID, uID int64) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			p := params
			uniquePath, uParams, err := uniquify.ProcessForAccount(params.VideoPath, aID)
			if err != nil {
				log.Printf("Account %d: uniquify skipped (%v), uploading original", aID, err)
			} else {
				log.Printf("Account %d: uniquified — %s", aID, uParams)
				p.VideoPath = uniquePath
				defer os.Remove(uniquePath)
			}

			res := UploadToAccount(c, p, aID, uID)
			mu.Lock()
			results = append(results, res)
			mu.Unlock()
		}(client, acc.ID, uploadID)
	}

	wg.Wait()
	return results
}

func MoveToUploaded(videoPath, uploadedDir string) error {
	if err := os.MkdirAll(uploadedDir, 0755); err != nil {
		return err
	}
	base := filepath.Base(videoPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	dest := filepath.Join(uploadedDir, fmt.Sprintf("%s_%d%s", name, time.Now().Unix(), ext))
	return os.Rename(videoPath, dest)
}

func ListPendingVideos(pendingDir string) ([]string, error) {
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		return nil, err
	}
	videoExts := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true}
	var videos []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if videoExts[ext] {
			videos = append(videos, filepath.Join(pendingDir, e.Name()))
		}
	}
	return videos, nil
}
