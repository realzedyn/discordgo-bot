package events

import (
	"discord-bot/internal/logger"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (h *EventHandler) StartWorkers(s *discordgo.Session) {
	go h.tempAccessWorker(s)
}

func (h *EventHandler) tempAccessWorker(s *discordgo.Session) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		profiles, err := h.DB.GetAllProfiles()
		if err != nil {
			logger.Error("Worker: Failed to fetch profiles: %v", err)
			continue
		}

		now := time.Now()
		for _, profile := range profiles {
			for _, access := range profile.TempAccesses {
				if now.After(access.ExpiresAt) {

					if access.Type == "role" {
						err := s.GuildMemberRoleRemove(access.GuildID, profile.UserID, access.TargetID)
						if err != nil {
							logger.Error("Worker: Failed to remove expired role %s from user %s: %v", access.TargetID, profile.UserID, err)
						} else {
							logger.Info("Worker: Removed expired role %s from user %s", access.TargetID, profile.UserID)

							dmChannel, err := s.UserChannelCreate(profile.UserID)
							if err == nil {
								s.ChannelMessageSend(dmChannel.ID, fmt.Sprintf("⏰ **Bilgi:** Geçici rolünüzün süresi doldu ve kaldırıldı: <@&%s>", access.TargetID))
							}
						}
					}

					err = h.DB.RemoveTempAccess(profile.UserID, access.TargetID)
					if err != nil {
						logger.Error("Worker: Failed to remove expired access from DB for %s: %v", profile.UserID, err)
					}
				}
			}
		}
	}
}
