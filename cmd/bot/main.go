package main

import (
	"discord-bot/internal/bot"
	"discord-bot/internal/config"
	"discord-bot/internal/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	cfg := config.Load()

	b := bot.New(cfg)
	if b == nil {
		logger.Error("Initialization failed")
		os.Exit(1)
	}

	err := b.Start()
	if err != nil {
		logger.Error("Failed to start bot: %v", err)
		os.Exit(1)
	}

	logger.Success("Bot is online! Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Warn("Shutting down...")
	b.Stop()
}
