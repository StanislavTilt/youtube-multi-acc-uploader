package cmd

import (
	"fmt"
	"log"

	"youtube-uploader/internal/db"
)

func RunPresetCreate(name, titleTpl, description, tagsStr, categoryID, privacy string) {
	if name == "" {
		log.Fatal("Preset name is required")
	}
	if titleTpl == "" {
		titleTpl = "{filename}"
	}
	if categoryID == "" {
		categoryID = "22"
	}
	if privacy == "" {
		privacy = "public"
	}

	tags := parseTags(tagsStr)
	id, err := db.InsertPreset(name, titleTpl, description, tags, categoryID, privacy)
	if err != nil {
		log.Fatalf("Create preset: %v", err)
	}
	fmt.Printf("Preset '%s' created (id: %d)\n", name, id)
}

func RunPresetList() {
	presets, err := db.ListPresets()
	if err != nil {
		log.Fatalf("List presets: %v", err)
	}
	if len(presets) == 0 {
		fmt.Println("No presets. Create one with: preset create --name <name>")
		return
	}
	fmt.Printf("%-4s %-15s %-25s %-10s %-10s %s\n", "ID", "NAME", "TITLE TEMPLATE", "CATEGORY", "PRIVACY", "TAGS")
	for _, p := range presets {
		fmt.Printf("%-4d %-15s %-25s %-10s %-10s %v\n", p.ID, p.Name, p.TitleTemplate, p.CategoryID, p.Privacy, p.Tags)
	}
}
