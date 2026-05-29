package config

import (
	"discord-bot/internal/logger"
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Token            string            `json:"-"`
	Prefix           string            `json:"prefix"`
	Status           string            `json:"status"`
	MongoURI         string            `json:"-"`
	DBName           string            `json:"-"`
	Admins           []string          `json:"admins"`
	StaffRoles       map[string]string `json:"staff_roles"`
	ReportLogChannel string            `json:"report_log_channel"`
	BoosterRole      string            `json:"booster_role"`
}

func Load() *Config {

	err := godotenv.Load()
	if err != nil {
		logger.Warn(".env file not found, will use system environment variables")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		logger.Error("DISCORD_TOKEN not found in environment")
		os.Exit(1)
	}

	file, err := os.Open("config.json")
	if err != nil {
		logger.Error("Failed to open config.json: %v", err)
		os.Exit(1)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		logger.Error("Failed to decode config.json: %v", err)
		os.Exit(1)
	}

	cfg.Token = token
	cfg.MongoURI = os.Getenv("MONGO_URI")
	cfg.DBName = os.Getenv("DB_NAME")

	if cfg.MongoURI == "" {
		cfg.MongoURI = "mongodb://localhost:27017"
	}
	if cfg.DBName == "" {
		cfg.DBName = "discord_bot"
	}

	return &cfg
}
