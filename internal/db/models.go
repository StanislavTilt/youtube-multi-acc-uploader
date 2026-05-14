package db

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Account struct {
	ID		int64		`json:"id"`
	ClientID	string		`json:"clientId"`
	ClientSecret	string		`json:"-"`
	Name		string		`json:"name"`
	CreatedAt	time.Time	`json:"createdAt"`
	DeletedAt	*time.Time	`json:"deletedAt,omitempty"`
}

func InsertAccount(clientID, clientSecret string) (int64, error) {
	res, err := DB.Exec(
		"INSERT INTO accounts (client_id, client_secret) VALUES (?, ?)",
		clientID, clientSecret,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListAccounts() ([]Account, error) {
	rows, err := DB.Query(
		"SELECT id, client_id, client_secret, name, created_at FROM accounts WHERE deleted_at IS NULL ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.ClientID, &a.ClientSecret, &a.Name, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func ListAllAccounts() ([]Account, error) {
	rows, err := DB.Query(
		"SELECT id, client_id, client_secret, name, created_at, deleted_at FROM accounts ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.ClientID, &a.ClientSecret, &a.Name, &a.CreatedAt, &a.DeletedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func GetAccount(id int64) (*Account, error) {
	var a Account
	err := DB.QueryRow(
		"SELECT id, client_id, client_secret, name, created_at FROM accounts WHERE id = ? AND deleted_at IS NULL", id,
	).Scan(&a.ID, &a.ClientID, &a.ClientSecret, &a.Name, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func SoftDeleteAccount(id int64) error {
	_, err := DB.Exec("UPDATE accounts SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL", id)
	return err
}

func HardDeleteAccount(id int64) error {
	DB.Exec("DELETE FROM tokens WHERE account_id = ?", id)
	DB.Exec("DELETE FROM uploads WHERE account_id = ?", id)
	_, err := DB.Exec("DELETE FROM accounts WHERE id = ?", id)
	return err
}

func RestoreAccount(id int64) error {
	_, err := DB.Exec("UPDATE accounts SET deleted_at = NULL WHERE id = ?", id)
	return err
}

type Token struct {
	ID		int64		`json:"id"`
	AccountID	int64		`json:"accountId"`
	AccessToken	string		`json:"-"`
	RefreshToken	string		`json:"-"`
	TokenType	string		`json:"tokenType"`
	Expiry		time.Time	`json:"expiry"`
	ChannelID	string		`json:"channelId"`
	ChannelName	string		`json:"channelName"`
}

func UpsertToken(accountID int64, accessToken, refreshToken, tokenType string, expiry time.Time, channelID, channelName string) error {
	_, err := DB.Exec(`
		INSERT INTO tokens (account_id, access_token, refresh_token, token_type, expiry, channel_id, channel_name)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(account_id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_type = excluded.token_type,
			expiry = excluded.expiry,
			channel_id = excluded.channel_id,
			channel_name = excluded.channel_name
	`, accountID, accessToken, refreshToken, tokenType, expiry, channelID, channelName)
	return err
}

func GetToken(accountID int64) (*Token, error) {
	var t Token
	err := DB.QueryRow(
		"SELECT id, account_id, access_token, refresh_token, token_type, expiry, channel_id, channel_name FROM tokens WHERE account_id = ?",
		accountID,
	).Scan(&t.ID, &t.AccountID, &t.AccessToken, &t.RefreshToken, &t.TokenType, &t.Expiry, &t.ChannelID, &t.ChannelName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

type Upload struct {
	ID		int64		`json:"id"`
	AccountID	int64		`json:"accountId"`
	VideoPath	string		`json:"videoPath"`
	YoutubeID	string		`json:"youtubeId"`
	Title		string		`json:"title"`
	Description	string		`json:"description"`
	Tags		[]string	`json:"tags"`
	Status		string		`json:"status"`
	Error		string		`json:"error,omitempty"`
	CreatedAt	time.Time	`json:"createdAt"`
}

func InsertUpload(accountID int64, videoPath, title, description string, tags []string) (int64, error) {
	tagsJSON, _ := json.Marshal(tags)
	res, err := DB.Exec(
		"INSERT INTO uploads (account_id, video_path, title, description, tags) VALUES (?, ?, ?, ?, ?)",
		accountID, videoPath, title, description, string(tagsJSON),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateUploadStatus(id int64, status, youtubeID, errMsg string) error {
	_, err := DB.Exec(
		"UPDATE uploads SET status = ?, youtube_id = ?, error = ? WHERE id = ?",
		status, youtubeID, errMsg, id,
	)
	return err
}

func DeleteUpload(id int64) error {
	_, err := DB.Exec("DELETE FROM uploads WHERE id = ?", id)
	return err
}

func ListUploads(limit int) ([]Upload, error) {
	rows, err := DB.Query(
		"SELECT id, account_id, video_path, youtube_id, title, description, tags, status, error, created_at FROM uploads ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uploads []Upload
	for rows.Next() {
		var u Upload
		var tagsStr string
		if err := rows.Scan(&u.ID, &u.AccountID, &u.VideoPath, &u.YoutubeID, &u.Title, &u.Description, &tagsStr, &u.Status, &u.Error, &u.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsStr), &u.Tags)
		uploads = append(uploads, u)
	}
	return uploads, rows.Err()
}

type Preset struct {
	ID		int64		`json:"id"`
	Name		string		`json:"name"`
	TitleTemplate	string		`json:"titleTemplate"`
	Description	string		`json:"description"`
	Tags		[]string	`json:"tags"`
	CategoryID	string		`json:"categoryId"`
	Privacy		string		`json:"privacy"`
	CreatedAt	time.Time	`json:"createdAt"`
}

func InsertPreset(name, titleTpl, description string, tags []string, categoryID, privacy string) (int64, error) {
	tagsJSON, _ := json.Marshal(tags)
	res, err := DB.Exec(
		"INSERT INTO video_presets (name, title_template, description, tags, category_id, privacy) VALUES (?, ?, ?, ?, ?, ?)",
		name, titleTpl, description, string(tagsJSON), categoryID, privacy,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetPreset(name string) (*Preset, error) {
	var p Preset
	var tagsStr string
	err := DB.QueryRow(
		"SELECT id, name, title_template, description, tags, category_id, privacy, created_at FROM video_presets WHERE name = ?", name,
	).Scan(&p.ID, &p.Name, &p.TitleTemplate, &p.Description, &tagsStr, &p.CategoryID, &p.Privacy, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	json.Unmarshal([]byte(tagsStr), &p.Tags)
	return &p, err
}

func ListPresets() ([]Preset, error) {
	rows, err := DB.Query("SELECT id, name, title_template, description, tags, category_id, privacy, created_at FROM video_presets ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var presets []Preset
	for rows.Next() {
		var p Preset
		var tagsStr string
		if err := rows.Scan(&p.ID, &p.Name, &p.TitleTemplate, &p.Description, &tagsStr, &p.CategoryID, &p.Privacy, &p.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tagsStr), &p.Tags)
		presets = append(presets, p)
	}
	return presets, rows.Err()
}

type Schedule struct {
	ID		int64		`json:"id"`
	VideoPath	*string		`json:"videoPath"`
	PresetID	*int64		`json:"presetId"`
	Title		string		`json:"title"`
	Description	string		`json:"description"`
	Tags		[]string	`json:"tags"`
	AccountIDs	[]int64		`json:"accountIds"`
	CronExpr	string		`json:"cronExpr"`
	NextRunAt	*time.Time	`json:"nextRunAt"`
	LastRunAt	*time.Time	`json:"lastRunAt"`
	Enabled		bool		`json:"enabled"`
	CreatedAt	time.Time	`json:"createdAt"`
}

func InsertSchedule(cronExpr, title, description string, tags []string, accountIDs []int64, presetID *int64) (int64, error) {
	tagsJSON, _ := json.Marshal(tags)
	accountsJSON, _ := json.Marshal(accountIDs)
	var accountsStr *string
	if len(accountIDs) > 0 {
		s := string(accountsJSON)
		accountsStr = &s
	}
	res, err := DB.Exec(
		"INSERT INTO schedule (cron_expr, title, description, tags, account_ids, preset_id) VALUES (?, ?, ?, ?, ?, ?)",
		cronExpr, title, description, string(tagsJSON), accountsStr, presetID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListSchedules() ([]Schedule, error) {
	rows, err := DB.Query(
		"SELECT id, video_path, preset_id, title, description, tags, account_ids, cron_expr, next_run_at, last_run_at, enabled, created_at FROM schedule ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		var tagsStr, accountsStr sql.NullString
		if err := rows.Scan(&s.ID, &s.VideoPath, &s.PresetID, &s.Title, &s.Description, &tagsStr, &accountsStr, &s.CronExpr, &s.NextRunAt, &s.LastRunAt, &s.Enabled, &s.CreatedAt); err != nil {
			return nil, err
		}
		if tagsStr.Valid {
			json.Unmarshal([]byte(tagsStr.String), &s.Tags)
		}
		if accountsStr.Valid {
			json.Unmarshal([]byte(accountsStr.String), &s.AccountIDs)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func SetScheduleEnabled(id int64, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := DB.Exec("UPDATE schedule SET enabled = ? WHERE id = ?", v, id)
	return err
}

func UpdateScheduleLastRun(id int64, nextRun *time.Time) error {
	_, err := DB.Exec("UPDATE schedule SET last_run_at = CURRENT_TIMESTAMP, next_run_at = ? WHERE id = ?", nextRun, id)
	return err
}

type QuickTag struct {
	ID		int64	`json:"id"`
	Tag		string	`json:"tag"`
	UseCount	int	`json:"useCount"`
}

func ListQuickTags() ([]QuickTag, error) {
	rows, err := DB.Query("SELECT id, tag, use_count FROM quick_tags ORDER BY use_count DESC, tag ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []QuickTag
	for rows.Next() {
		var t QuickTag
		if err := rows.Scan(&t.ID, &t.Tag, &t.UseCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func InsertQuickTag(tag string) (int64, error) {
	res, err := DB.Exec("INSERT OR IGNORE INTO quick_tags (tag) VALUES (?)", tag)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func DeleteQuickTag(id int64) error {
	_, err := DB.Exec("DELETE FROM quick_tags WHERE id = ?", id)
	return err
}

func IncrementQuickTagUse(tags []string) {
	for _, t := range tags {
		DB.Exec("UPDATE quick_tags SET use_count = use_count + 1 WHERE tag = ?", t)
	}
}

func GetDueSchedules() ([]Schedule, error) {
	rows, err := DB.Query(
		"SELECT id, video_path, preset_id, title, description, tags, account_ids, cron_expr, next_run_at, last_run_at, enabled, created_at FROM schedule WHERE enabled = 1 AND (next_run_at IS NULL OR next_run_at <= CURRENT_TIMESTAMP)",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		var tagsStr, accountsStr sql.NullString
		if err := rows.Scan(&s.ID, &s.VideoPath, &s.PresetID, &s.Title, &s.Description, &tagsStr, &accountsStr, &s.CronExpr, &s.NextRunAt, &s.LastRunAt, &s.Enabled, &s.CreatedAt); err != nil {
			return nil, err
		}
		if tagsStr.Valid {
			json.Unmarshal([]byte(tagsStr.String), &s.Tags)
		}
		if accountsStr.Valid {
			json.Unmarshal([]byte(accountsStr.String), &s.AccountIDs)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}
