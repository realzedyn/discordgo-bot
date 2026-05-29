package state

import (
	"discord-bot/internal/database"
	"discord-bot/internal/logger"
	"discord-bot/internal/models"
	"encoding/json"
	"os"
)

type Manager struct {
	DB     *database.Database
	Tasks  []models.Task
	Badges []models.Badge
}

func NewManager(db *database.Database) *Manager {
	m := &Manager{
		DB: db,
	}
	m.LoadTasks("tasks.json")
	m.LoadBadges("badges.json")
	return m
}

func (m *Manager) LoadBadges(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Failed to open badges.json: %v", err)
		return
	}
	defer file.Close()

	var badges []models.Badge
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&badges)
	if err != nil {
		logger.Error("Failed to decode badges.json: %v", err)
		return
	}

	m.Badges = badges
	logger.Success("Loaded %d badges from %s", len(m.Badges), filePath)
}

func (m *Manager) LoadTasks(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Failed to open tasks.json: %v", err)
		return
	}
	defer file.Close()

	var tasks []models.Task
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&tasks)
	if err != nil {
		logger.Error("Failed to decode tasks.json: %v", err)
		return
	}

	m.Tasks = tasks
	logger.Success("Loaded %d tasks from %s", len(m.Tasks), filePath)
}

func (m *Manager) GetBadge(id string) models.Badge {
	for _, badge := range m.Badges {
		if badge.ID == id {
			return badge
		}
	}
	return models.Badge{}
}
