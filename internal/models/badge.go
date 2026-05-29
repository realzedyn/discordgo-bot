package models

type Badge struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconEmoji   string `json:"icon_emoji"`
	Category    string `json:"category"`
	Color       string `json:"color"`
}
