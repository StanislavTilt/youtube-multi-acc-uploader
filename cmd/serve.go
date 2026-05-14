package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"youtube-uploader/internal/config"
	"youtube-uploader/internal/db"
	"youtube-uploader/internal/logger"
	"youtube-uploader/internal/youtube"
)

var appLogger *logger.Logger

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonCreated(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func jsonAccepted(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(data)
}

func httpErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func parseID(r *http.Request) int64 {
	var id int64
	fmt.Sscanf(r.PathValue("id"), "%d", &id)
	return id
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func htmlPage(w http.ResponseWriter, color, body string) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html><body style="background:#0c0c0e;color:%s;font-family:sans-serif;display:flex;align-items:center;justify-content:center;height:100vh;flex-direction:column">%s</body></html>`, color, body)
}

type accountWithAuth struct {
	db.Account
	Authorized	bool	`json:"authorized"`
	ChannelName	string	`json:"channelName,omitempty"`
}

func handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := db.ListAllAccounts()
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	var result []accountWithAuth
	for _, a := range accounts {
		aa := accountWithAuth{Account: a}
		if tok, _ := db.GetToken(a.ID); tok != nil {
			aa.Authorized = true
			aa.ChannelName = tok.ChannelName
		}
		result = append(result, aa)
	}
	jsonOK(w, result)
}

func handleAddAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClientID	string	`json:"clientId"`
		ClientSecret	string	`json:"clientSecret"`
	}
	if err := decodeJSON(r, &req); err != nil {
		httpErr(w, 400, "invalid JSON")
		return
	}
	if req.ClientID == "" || req.ClientSecret == "" {
		httpErr(w, 400, "clientId and clientSecret are required")
		return
	}
	id, err := db.InsertAccount(req.ClientID, req.ClientSecret)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	if id == 0 {
		httpErr(w, 409, "account already exists")
		return
	}
	jsonCreated(w, map[string]any{"id": id, "status": "created"})
}

func handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	RunAccountDelete(r.PathValue("id"))
	jsonOK(w, map[string]string{"status": "deleted"})
}

func handleHardDeleteAccount(w http.ResponseWriter, r *http.Request) {
	id := parseID(r)
	if err := db.HardDeleteAccount(id); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]string{"status": "permanently deleted"})
}

func handleRestoreAccount(w http.ResponseWriter, r *http.Request) {
	RunAccountRestore(r.PathValue("id"))
	jsonOK(w, map[string]string{"status": "restored"})
}

func handleAuthAccount(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := parseID(r)
		acc, err := db.GetAccount(id)
		if err != nil || acc == nil {
			httpErr(w, 404, "account not found")
			return
		}
		authURL := youtube.GetAuthURL(cfg, acc, fmt.Sprintf("account_%d", id))
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

func handleAuthNew(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
			httpErr(w, 500, "GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set in .env")
			return
		}
		authURL := youtube.GetNewAccountAuthURL(cfg, "new_account")
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

func handleAuthAll(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := db.ListAccounts()
		if err != nil {
			httpErr(w, 500, err.Error())
			return
		}
		type authInfo struct {
			AccountID	int64	`json:"accountId"`
			AuthURL		string	`json:"authUrl"`
		}
		var result []authInfo
		for _, a := range accounts {
			if tok, _ := db.GetToken(a.ID); tok != nil {
				continue
			}
			url := youtube.GetAuthURL(cfg, &a, fmt.Sprintf("account_%d", a.ID))
			result = append(result, authInfo{AccountID: a.ID, AuthURL: url})
		}
		jsonOK(w, result)
	}
}

func handleOAuthCallback(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" {
			htmlPage(w, "#ef4444", "<h2>Error: no authorization code received</h2>")
			return
		}

		if state == "new_account" {
			accountID, err := youtube.ExchangeCodeNewAccount(cfg, code)
			if err != nil {
				appLogger.YouTube(logger.LogEntry{Level: "error", Message: "New account OAuth failed: " + err.Error()})
				htmlPage(w, "#ef4444", fmt.Sprintf("<h2>Authorization failed</h2><p style='color:#918e88'>%s</p>", err.Error()))
				return
			}
			appLogger.YouTube(logger.LogEntry{Level: "info", Message: "New account added via OAuth", AccountID: accountID})
			htmlPage(w, "#22c55e", fmt.Sprintf(
				`<h2 style="margin-bottom:12px">Account #%d added and authorized!</h2><p style="color:#918e88">Redirecting...</p><script>setTimeout(()=>window.location.href='/',1500)</script>`, accountID))
			return
		}

		var accountID int64
		fmt.Sscanf(state, "account_%d", &accountID)
		if accountID == 0 {
			htmlPage(w, "#ef4444", "<h2>Error: invalid state</h2>")
			return
		}

		acc, err := db.GetAccount(accountID)
		if err != nil || acc == nil {
			htmlPage(w, "#ef4444", "<h2>Error: account not found</h2>")
			return
		}

		if err := youtube.ExchangeCode(cfg, acc, code); err != nil {
			appLogger.YouTube(logger.LogEntry{Level: "error", Message: "OAuth exchange failed: " + err.Error(), AccountID: accountID})
			htmlPage(w, "#ef4444", fmt.Sprintf("<h2>Authorization failed</h2><p style='color:#918e88'>%s</p>", err.Error()))
			return
		}

		appLogger.YouTube(logger.LogEntry{Level: "info", Message: "OAuth authorized via web", AccountID: accountID})
		htmlPage(w, "#22c55e", fmt.Sprintf(
			`<h2 style="margin-bottom:12px">Account #%d authorized!</h2><p style="color:#918e88">Redirecting...</p><script>setTimeout(()=>window.location.href='/',1500)</script>`, accountID))
	}
}

func handleListQuickTags(w http.ResponseWriter, r *http.Request) {
	tags, err := db.ListQuickTags()
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonOK(w, tags)
}

func handleAddQuickTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tag string `json:"tag"`
	}
	if err := decodeJSON(r, &req); err != nil || req.Tag == "" {
		httpErr(w, 400, "tag is required")
		return
	}
	id, err := db.InsertQuickTag(req.Tag)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonCreated(w, map[string]any{"id": id, "tag": req.Tag})
}

func handleDeleteQuickTag(w http.ResponseWriter, r *http.Request) {
	db.DeleteQuickTag(parseID(r))
	jsonOK(w, map[string]string{"status": "deleted"})
}

func handleListPending(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		videos, err := youtube.ListPendingVideos(cfg.VideosPending)
		if err != nil {
			httpErr(w, 500, err.Error())
			return
		}
		var names []string
		for _, v := range videos {
			names = append(names, filepath.Base(v))
		}
		jsonOK(w, names)
	}
}

func handleFileUpload(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(5 << 30)
		file, header, err := r.FormFile("video")
		if err != nil {
			httpErr(w, 400, "video file is required")
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		allowed := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true}
		if !allowed[ext] {
			httpErr(w, 400, "unsupported format: "+ext)
			return
		}

		os.MkdirAll(cfg.VideosPending, 0755)
		dst, err := os.Create(filepath.Join(cfg.VideosPending, header.Filename))
		if err != nil {
			httpErr(w, 500, "create file: "+err.Error())
			return
		}
		defer dst.Close()
		io.Copy(dst, file)
		jsonCreated(w, map[string]string{"filename": header.Filename, "status": "uploaded"})
	}
}

func handleYouTubeUpload(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Filename	string		`json:"filename"`
			Title		string		`json:"title"`
			Description	string		`json:"description"`
			Tags		[]string	`json:"tags"`
			AccountIDs	[]int64		`json:"accountIds"`
			Privacy		string		`json:"privacy"`
			ScheduleAt	string		`json:"scheduleAt"`
		}
		if err := decodeJSON(r, &req); err != nil {
			httpErr(w, 400, "invalid JSON")
			return
		}
		if req.Filename == "" || req.Title == "" {
			httpErr(w, 400, "filename and title are required")
			return
		}

		videoPath := filepath.Join(cfg.VideosPending, req.Filename)
		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			httpErr(w, 404, "video not found in pending/")
			return
		}

		accounts, err := db.ListAccounts()
		if err != nil {
			httpErr(w, 500, err.Error())
			return
		}
		if len(req.AccountIDs) > 0 {
			idSet := make(map[int64]bool)
			for _, id := range req.AccountIDs {
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
		if len(accounts) == 0 {
			httpErr(w, 400, "no accounts available")
			return
		}

		privacy := req.Privacy
		if privacy == "" {
			privacy = "public"
		}
		params := youtube.UploadParams{
			VideoPath:	videoPath, Title: req.Title, Description: req.Description,
			Tags:	req.Tags, CategoryID: "22", Privacy: privacy,
		}

		if req.ScheduleAt != "" {
			tagsJSON, _ := json.Marshal(req.Tags)
			accountsJSON, _ := json.Marshal(req.AccountIDs)
			var acctStr *string
			if len(req.AccountIDs) > 0 {
				s := string(accountsJSON)
				acctStr = &s
			}
			_, err := db.DB.Exec(`INSERT INTO schedule (video_path, title, description, tags, account_ids, cron_expr, next_run_at, enabled)
				VALUES (?, ?, ?, ?, ?, ?, ?, 1)`,
				videoPath, req.Title, req.Description, string(tagsJSON), acctStr, "manual", req.ScheduleAt)
			if err != nil {
				httpErr(w, 500, err.Error())
				return
			}
			jsonCreated(w, map[string]string{"status": "scheduled", "scheduledAt": req.ScheduleAt})
			return
		}

		db.IncrementQuickTagUse(req.Tags)

		go func() {
			appLogger.YouTube(logger.LogEntry{Level: "info", Message: fmt.Sprintf("Upload started — title: %s, tags: %v, desc: %s", req.Title, req.Tags, req.Description), VideoFile: req.Filename})

			var authAccounts []db.Account
			for _, a := range accounts {
				if tok, _ := db.GetToken(a.ID); tok != nil {
					authAccounts = append(authAccounts, a)
				} else {
					appLogger.YouTube(logger.LogEntry{Level: "warn", Message: "Skipped — no token", AccountID: a.ID, VideoFile: req.Filename})
				}
			}
			if len(authAccounts) == 0 {
				appLogger.YouTube(logger.LogEntry{Level: "error", Message: "No authorized accounts", VideoFile: req.Filename})
				return
			}

			results := youtube.UploadToAll(cfg, params, authAccounts, cfg.MaxConcurrency)
			success := 0
			for _, res := range results {
				if res.Error != nil {
					appLogger.YouTube(logger.LogEntry{Level: "error", Message: res.Error.Error(), AccountID: res.AccountID, VideoFile: req.Filename})
				} else {
					appLogger.YouTube(logger.LogEntry{Level: "info", Message: "Upload success", AccountID: res.AccountID, VideoFile: req.Filename, YoutubeID: res.YoutubeID})
					success++
				}
			}
			if success > 0 && success == len(results) {
				youtube.MoveToUploaded(videoPath, cfg.VideosUploaded)
			}
			appLogger.YouTube(logger.LogEntry{Level: "info", Message: fmt.Sprintf("Complete: %d/%d success", success, len(results)), VideoFile: req.Filename})
		}()

		jsonAccepted(w, map[string]any{"status": "uploading", "accounts": len(accounts), "filename": req.Filename})
	}
}

func handleDeleteUpload(w http.ResponseWriter, r *http.Request) {
	if err := db.DeleteUpload(parseID(r)); err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

func handleListUploads(w http.ResponseWriter, r *http.Request) {
	uploads, err := db.ListUploads(100)
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonOK(w, uploads)
}

func handleListPresets(w http.ResponseWriter, r *http.Request) {
	presets, err := db.ListPresets()
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonOK(w, presets)
}

func handleListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := db.ListSchedules()
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	jsonOK(w, schedules)
}

func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "admin"
	}
	logPath := filepath.Join("./logs", category+"-"+date+".json")
	data, err := os.ReadFile(logPath)
	if err != nil {
		jsonOK(w, []any{})
		return
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var entries []json.RawMessage
	for _, line := range lines {
		if line != "" {
			entries = append(entries, json.RawMessage(line))
		}
	}
	jsonOK(w, entries)
}

func handleGetLogDates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	if category == "" {
		category = "admin"
	}
	entries, _ := os.ReadDir("./logs")
	prefix := category + "-"
	var dates []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".json") {
			dates = append(dates, strings.TrimSuffix(strings.TrimPrefix(e.Name(), prefix), ".json"))
		}
	}
	jsonOK(w, dates)
}

func RunServe(cfg *config.Config) {
	var err error
	appLogger, err = logger.New("./logs")
	if err != nil {
		log.Fatalf("Init logger: %v", err)
	}
	defer appLogger.Close()
	appLogger.Info("Server starting on " + cfg.Host + ":" + cfg.Port)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/accounts", handleListAccounts)
	mux.HandleFunc("POST /api/accounts", handleAddAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}", handleDeleteAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}/permanent", handleHardDeleteAccount)
	mux.HandleFunc("POST /api/accounts/{id}/restore", handleRestoreAccount)

	mux.HandleFunc("GET /api/accounts/{id}/auth", handleAuthAccount(cfg))
	mux.HandleFunc("GET /api/auth/new", handleAuthNew(cfg))
	mux.HandleFunc("GET /api/auth/all", handleAuthAll(cfg))
	mux.HandleFunc("GET /callback", handleOAuthCallback(cfg))

	mux.HandleFunc("GET /api/quick-tags", handleListQuickTags)
	mux.HandleFunc("POST /api/quick-tags", handleAddQuickTag)
	mux.HandleFunc("DELETE /api/quick-tags/{id}", handleDeleteQuickTag)

	mux.HandleFunc("GET /api/videos/pending", handleListPending(cfg))
	mux.HandleFunc("POST /api/videos/upload", handleFileUpload(cfg))
	mux.HandleFunc("POST /api/upload", handleYouTubeUpload(cfg))

	mux.HandleFunc("GET /api/uploads", handleListUploads)
	mux.HandleFunc("DELETE /api/uploads/{id}", handleDeleteUpload)
	mux.HandleFunc("GET /api/presets", handleListPresets)
	mux.HandleFunc("GET /api/schedules", handleListSchedules)

	mux.HandleFunc("GET /api/logs", handleGetLogs)
	mux.HandleFunc("GET /api/logs/dates", handleGetLogDates)

	mux.Handle("GET /", http.FileServer(http.Dir("internal/web/static")))

	handler := appLogger.Middleware(mux)
	addr := cfg.Host + ":" + cfg.Port
	log.Printf("Web panel: http://localhost:%s", cfg.Port)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
