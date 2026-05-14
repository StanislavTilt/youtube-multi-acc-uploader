package cmd

import (
	"fmt"
	"log"
	"strconv"

	"youtube-uploader/internal/db"
)

func RunScheduleAdd(cronExpr, title, description, tagsStr, presetName string) {
	if cronExpr == "" {
		log.Fatal("Cron expression is required (--cron)")
	}

	var presetID *int64
	if presetName != "" {
		preset, err := db.GetPreset(presetName)
		if err != nil {
			log.Fatalf("Get preset: %v", err)
		}
		if preset == nil {
			log.Fatalf("Preset '%s' not found", presetName)
		}
		presetID = &preset.ID
		if title == "" {
			title = preset.TitleTemplate
		}
		if description == "" {
			description = preset.Description
		}
	}

	tags := parseTags(tagsStr)

	id, err := db.InsertSchedule(cronExpr, title, description, tags, nil, presetID)
	if err != nil {
		log.Fatalf("Create schedule: %v", err)
	}
	fmt.Printf("Schedule #%d created: %s\n", id, cronExpr)
}

func RunScheduleList() {
	schedules, err := db.ListSchedules()
	if err != nil {
		log.Fatalf("List schedules: %v", err)
	}
	if len(schedules) == 0 {
		fmt.Println("No schedules. Add one with: schedule add --cron \"0 9 * * *\"")
		return
	}
	fmt.Printf("%-4s %-20s %-8s %-20s %s\n", "ID", "CRON", "ENABLED", "LAST RUN", "TITLE")
	for _, s := range schedules {
		enabled := "yes"
		if !s.Enabled {
			enabled = "no"
		}
		lastRun := "never"
		if s.LastRunAt != nil {
			lastRun = s.LastRunAt.Format("2006-01-02 15:04")
		}
		fmt.Printf("%-4d %-20s %-8s %-20s %s\n", s.ID, s.CronExpr, enabled, lastRun, s.Title)
	}
}

func RunScheduleToggle(idStr string, enable bool) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid schedule ID: %s", idStr)
	}
	if err := db.SetScheduleEnabled(id, enable); err != nil {
		log.Fatalf("Update schedule: %v", err)
	}
	action := "disabled"
	if enable {
		action = "enabled"
	}
	fmt.Printf("Schedule #%d %s\n", id, action)
}
