package events

import (
	"discord-bot/internal/commands"
	"discord-bot/internal/config"
	"discord-bot/internal/database"
	"discord-bot/internal/logger"
	"discord-bot/internal/models"
	"discord-bot/internal/state"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type EventHandler struct {
	Registry    *commands.Registry
	AppRegistry *commands.ApplicationRegistry
	DB          *database.Database
	State       *state.Manager
	Config      *config.Config
}

func NewEventHandler(r *commands.Registry, ar *commands.ApplicationRegistry, db *database.Database, s *state.Manager, cfg *config.Config) *EventHandler {
	return &EventHandler{
		Registry:    r,
		AppRegistry: ar,
		DB:          db,
		State:       s,
		Config:      cfg,
	}
}

func (h *EventHandler) OnReady(s *discordgo.Session, r *discordgo.Ready) {
	logger.Success("Bot logged in as %s#%s", r.User.Username, r.User.Discriminator)

	var cmds []*discordgo.ApplicationCommand
	for _, cmd := range h.AppRegistry.Commands {
		cmds = append(cmds, cmd.Definition())
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", cmds)
	if err != nil {
		logger.Error("Failed to sync application commands: %v", err)
	}
}

func (h *EventHandler) OnGuildMemberUpdate(s *discordgo.Session, m *discordgo.GuildMemberUpdate) {

	roleBadgeMap := make(map[string]string)
	for cat, roleID := range h.Config.StaffRoles {
		var badgeID string
		switch cat {
		case "sharer":
			badgeID = "sharer_badge"
		case "trial_moderator":
			badgeID = "trial_mod_badge"
		case "moderator":
			badgeID = "moderator_badge"
		case "admin":
			badgeID = "admin_badge"
		}
		if badgeID != "" && roleID != "" {
			roleBadgeMap[roleID] = badgeID
		}
	}

	if h.Config.BoosterRole != "" {
		roleBadgeMap[h.Config.BoosterRole] = "supporter_badge"
	}

	for _, roleID := range m.Roles {
		if badgeID, ok := roleBadgeMap[roleID]; ok {
			err := h.DB.AddBadge(m.User.ID, badgeID)
			logger.Debug("Added badge %s for user %s", badgeID, m.User.ID)
			if err != nil {
				logger.Error("Failed to sync badge %s for user %s: %v", badgeID, m.User.ID, err)
			}
		}
	}

	profile, err := h.DB.GetProfile(m.User.ID)
	if err == nil {
		for roleID, badgeID := range roleBadgeMap {
			hasBadge := slices.Contains(profile.Badges, badgeID)

			if hasBadge {

				hasRole := slices.Contains(m.Roles, roleID)

				if !hasRole {
					err := h.DB.RemoveBadge(m.User.ID, badgeID)
					logger.Debug("Removed badge %s for user %s", badgeID, m.User.ID)
					if err != nil {
						logger.Error("Failed to remove synced badge %s for user %s: %v", badgeID, m.User.ID, err)
					}
				}
			}
		}
	}
}

func (h *EventHandler) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	shares := 0
	if len(m.Attachments) > 0 || strings.Contains(m.Content, "http") {
		shares = 1
	}

	profile, _ := h.DB.GetProfile(m.Author.ID)
	xpToAdd := 2
	if !profile.LastMessageAt.IsZero() && time.Since(profile.LastMessageAt) < 30*time.Second {
		xpToAdd = 1
	}

	err := h.DB.IncrementStats(m.Author.ID, 1, shares)
	if err != nil {
		logger.Error("Failed to update profile for %s: %v", m.Author.ID, err)
	}
	h.DB.AddXP(m.Author.ID, xpToAdd)
	h.DB.UpdateLastMessageAt(m.Author.ID)

	go h.checkTasks(s, m.Author.ID, m.GuildID)

	if !strings.HasPrefix(m.Content, h.Registry.Prefix) {
		return
	}

	args := strings.Fields(m.Content[len(h.Registry.Prefix):])
	if len(args) == 0 {
		return
	}

	cmdName := strings.ToLower(args[0])
	if cmd, ok := h.Registry.Commands[cmdName]; ok {
		ctx := &commands.Context{
			Session: s,
			Message: m,
			Args:    args[1:],
			DB:      h.DB,
			State:   h.State,
			Config:  h.Config,
		}

		logger.Info("Executing command: %s (User: %s)", cmdName, m.Author.Username)

		if err := cmd.Execute(ctx); err != nil {
			logger.Error("Failed to execute command (%s): %v", cmdName, err)
		}
	}
}

func (h *EventHandler) checkTasks(s *discordgo.Session, userID, guildID string) {
	profile, err := h.DB.GetProfile(userID)
	if err != nil {
		return
	}

	now := time.Now()
	for _, task := range h.State.Tasks {

		isCompleted := false
		for _, p := range profile.Tasks {
			if p.TaskID == task.ID {
				if task.Category == "daily" {
					if p.Completed && p.LastCompleted.Year() == now.Year() && p.LastCompleted.YearDay() == now.YearDay() {
						isCompleted = true
					}
				} else {
					if p.Completed && !task.IsRepeatable {
						isCompleted = true
					}
				}
				break
			}
		}
		if isCompleted {
			continue
		}

		met := false
		req := task.Requirements

		msgCount := profile.MessageCount
		shrCount := profile.ShareCount
		if task.Category == "daily" {
			msgCount = profile.DailyMessageCount
			shrCount = profile.DailyShareCount
		}

		var currentStat int
		switch req.Type {
		case "message_count":
			currentStat = msgCount
		case "share_count":
			currentStat = shrCount
		}

		var baseValue int
		for _, p := range profile.Tasks {
			if p.TaskID == task.ID {
				if task.Category == "daily" {
					if p.LastCompleted.Year() == now.Year() && p.LastCompleted.YearDay() == now.YearDay() {
						baseValue = p.CurrentValue
					}
				} else {
					baseValue = p.CurrentValue
				}
				break
			}
		}

		progress := currentStat - baseValue
		if progress < 0 {
			progress = 0
		}

		if progress >= req.TargetValue {
			met = true
		}

		if met {
			logger.Info("User %s completed task: %s", userID, task.Name)
			err := h.DB.CompleteTask(userID, task.ID, currentStat, task.IsRepeatable)
			if err != nil {
				logger.Error("Failed to mark task as completed: %v", err)
				continue
			}

			for _, reward := range task.Rewards {
				h.grantReward(s, userID, guildID, reward, task.ID)
			}

			var builder strings.Builder

			for _, reward := range task.Rewards {
				switch reward.Type {

				case "badge":
					badge := h.State.GetBadge(reward.Value)
					fmt.Fprintf(&builder, "- **Rozet** - %s\n", badge.Name)

				case "temp_access":
					fmt.Fprintf(&builder, "- **Geçici Erişim** - <@&%s> (%s)\n", reward.Value, reward.Duration)

				default:
					fmt.Fprintf(&builder, "- **%s** - %s\n",
						strings.ToUpper(reward.Type),
						reward.Value,
					)
				}
			}

			var fieldName string = "Ödül"
			var fieldValue string = builder.String()

			if len(task.Rewards) > 1 {
				fieldName = "Ödüller"
			}

			var fields = []*discordgo.MessageEmbedField{
				{
					Name:  fieldName,
					Value: fieldValue,
				},
				{
					Name:  "\u200b",
					Value: "Ödüller profilinize eklendi. `!profile` ile görüntüleyin.",
				},
			}

			dmChannel, err := s.UserChannelCreate(userID)
			if err == nil {
				embed := &discordgo.MessageEmbed{
					Title:       "Görev Tamamlandı!",
					Description: fmt.Sprintf("**%s** görevini başarıyla tamamladın!", task.Name),
					Color:       0xFFD700,
					Fields:      fields,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Harika iş çıkardın!",
					},
				}
				s.ChannelMessageSendEmbed(dmChannel.ID, embed)
			}
		}
	}
}

func (h *EventHandler) OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		data := i.ApplicationCommandData()
		if cmd, ok := h.AppRegistry.Commands[data.Name]; ok {
			ctx := &commands.ApplicationContext{
				Session:     s,
				Interaction: i,
				DB:          h.DB,
				State:       h.State,
				Config:      h.Config,
			}

			logger.Info("Executing application command: %s (User: %s)", data.Name, i.Member.User.Username)
			if err := cmd.Execute(ctx); err != nil {
				logger.Error("Failed to execute application command (%s): %v", data.Name, err)
			}
		}
		return
	}

	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	customID := data.CustomID
	parts := strings.Split(customID, "_")
	if len(parts) < 3 {
		return
	}

	prefix := parts[0]

	category := parts[2]

	var commanderID, targetID string
	if prefix != "staff" {
		if len(parts) < 4 {
			return
		}
		commanderID = parts[3]
		targetID = commanderID
		if len(parts) >= 5 {
			targetID = parts[4]
		}
	}

	logger.Info("Executing interaction: %s (User: %s)", customID, i.Member.User.Username)

	if prefix != "staff" && i.Member.User.ID != commanderID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Bu menüyü sadece komutu kullanan kişi kontrol edebilir.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var profile *models.UserProfile
	var err error

	if prefix != "staff" {
		profile, err = h.DB.GetProfile(targetID)
		if err != nil {
			logger.Error("Failed to get profile for interaction: %v", err)
			return
		}
	}

	var embed *discordgo.MessageEmbed
	var categoryOrder []string

	if prefix == "tasks" {
		categories := make(map[string][]models.Task)
		for _, task := range h.State.Tasks {
			if _, ok := categories[task.Category]; !ok {
				categoryOrder = append(categoryOrder, task.Category)
			}
			categories[task.Category] = append(categories[task.Category], task)
		}

		targetUser, err := s.User(targetID)
		if err != nil {
			logger.Error("Failed to fetch target user for interaction: %v", err)
			return
		}
		embed = h.createTasksEmbed(profile, category, categories[category], targetUser, commanderID)
	} else if prefix == "badges" {
		categories := make(map[string][]models.Badge)
		for _, badge := range h.State.Badges {
			if _, ok := categories[badge.Category]; !ok {
				categoryOrder = append(categoryOrder, badge.Category)
			}
			categories[badge.Category] = append(categories[badge.Category], badge)
		}

		targetUser, err := s.User(targetID)
		if err != nil {
			logger.Error("Failed to fetch target user for interaction: %v", err)
			return
		}
		embed = h.createBadgesEmbed(profile, category, categories[category], targetUser, commanderID)
	} else if prefix == "profile" {
		user, err := s.User(targetID)
		if err != nil {
			logger.Error("Failed to fetch user for profile view: %v", err)
			return
		}

		member, err := s.GuildMember(i.GuildID, targetID)
		if err != nil {
			logger.Error("Failed to fetch member for profile view: %v", err)
			return
		}

		embed = h.createProfileEmbed(profile, user, member)
	} else if prefix == "staff" {
		categoryOrder = []string{"admin", "moderator", "trial_moderator", "sharer"}
		embed = h.createStaffEmbed(s, i.GuildID, category)
	}

	if embed == nil {
		return
	}

	components := h.createCategoryButtons(prefix, categoryOrder, category, commanderID, targetID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

var categoryTranslations = map[string]string{
	"milestones":   "Ana Görevler",
	"daily":        "Günlük Görevler",
	"staff":        "Yetkili",
	"achievements": "Başarımlar",
	"special":      "Özel",
}

func translateCategory(cat string) string {
	if translated, ok := categoryTranslations[cat]; ok {
		return translated
	}
	return strings.Title(cat)
}

func (h *EventHandler) createTasksEmbed(profile *models.UserProfile, category string, tasks []models.Task, targetUser *discordgo.User, commanderID string) *discordgo.MessageEmbed {
	var builder strings.Builder

	if category == "daily" {
		builder.WriteString("> 🕒 *Günlük görevler her gece 00:00'da sıfırlanmaktadır. Günde 1 kez yapılabilir.*\n\n")
	}

	now := time.Now()
	for _, task := range tasks {
		status := "⭕"
		progress := 0

		found := false
		for _, p := range profile.Tasks {
			if p.TaskID == task.ID {
				found = true

				isCompleted := false
				if task.Category == "daily" {
					if p.Completed && p.LastCompleted.Year() == now.Year() && p.LastCompleted.YearDay() == now.YearDay() {
						isCompleted = true
					}
				} else {
					if p.Completed {
						isCompleted = true
					}
				}

				if isCompleted {
					status = "✅"
					progress = task.Requirements.TargetValue
				} else {
					msgCount := profile.MessageCount
					shrCount := profile.ShareCount
					if task.Category == "daily" {
						msgCount = profile.DailyMessageCount
						shrCount = profile.DailyShareCount
					}

					var currentStat int
					switch task.Requirements.Type {
					case "message_count":
						currentStat = msgCount
					case "share_count":
						currentStat = shrCount
					}

					baseValue := 0
					if task.Category == "daily" {
						if p.LastCompleted.Year() == now.Year() && p.LastCompleted.YearDay() == now.YearDay() {
							baseValue = p.CurrentValue
						}
					} else {
						baseValue = p.CurrentValue
					}

					progress = currentStat - baseValue
					if progress < 0 {
						progress = 0
					}
				}
				break
			}
		}

		if !found {
			msgCount := profile.MessageCount
			shrCount := profile.ShareCount
			if task.Category == "daily" {
				msgCount = profile.DailyMessageCount
				shrCount = profile.DailyShareCount
			}

			switch task.Requirements.Type {
			case "message_count":
				progress = msgCount
			case "share_count":
				progress = shrCount
			}
		}

		if progress >= task.Requirements.TargetValue {
			progress = task.Requirements.TargetValue
		}

		builder.WriteString(fmt.Sprintf("%s **%s**\n", status, task.Name))
		builder.WriteString(fmt.Sprintf("└ *%s*\n", task.Description))
		builder.WriteString(fmt.Sprintf("└ %s `%d/%d`\n", h.generateProgressBar(progress, task.Requirements.TargetValue), progress, task.Requirements.TargetValue))

		var rewards []string
		for _, r := range task.Rewards {
			switch r.Type {
			case "badge":
				badge := h.State.GetBadge(r.Value)
				rewards = append(rewards, fmt.Sprintf("`rozet:%s`", strings.ToLower(badge.Name)))
			case "temp_access":
				rewards = append(rewards, fmt.Sprintf("`geçici_erişim:%s (%s)`", r.Value, r.Duration))
			default:
				rewards = append(rewards, fmt.Sprintf("`%s:%s`", r.Type, r.Value))
			}
		}
		builder.WriteString(fmt.Sprintf("└ Ödüller: %s\n\n", strings.Join(rewards, ", ")))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📝 Görevler - %s", translateCategory(category)),
		Description: builder.String(),
		Color:       0x4CAF50,
	}

	if targetUser.ID != commanderID {
		displayName := targetUser.Username
		if targetUser.GlobalName != "" {
			displayName = fmt.Sprintf("%s (%s)", targetUser.GlobalName, targetUser.Username)
		}
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    displayName,
			IconURL: targetUser.AvatarURL("128"),
		}
	}

	return embed
}

func (h *EventHandler) createBadgesEmbed(profile *models.UserProfile, category string, badges []models.Badge, targetUser *discordgo.User, commanderID string) *discordgo.MessageEmbed {
	var builder strings.Builder

	for _, badge := range badges {
		status := "🔒"
		for _, bID := range profile.Badges {
			if bID == badge.ID {
				status = "✨"
				break
			}
		}

		builder.WriteString(fmt.Sprintf("%s %s **%s**\n", status, badge.IconEmoji, badge.Name))
		builder.WriteString(fmt.Sprintf("└ *%s*\n\n", badge.Description))
	}

	var footerText string

	if category == "staff" {
		footerText = "Bu rozetler sadece yetkililere tanımlanır."
	} else if category == "special" {
		footerText = "Bu rozetler yöneticiler tarafından gereksinimleri karşılayanlara tanımlanır."
	} else {
		footerText = "Bu rozetleri kazanmak için görevleri tamamla. - !tasks"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🏆 Rozetler - %s", translateCategory(category)),
		Description: builder.String(),
		Color:       0xFFD700,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText,
		},
	}

	if targetUser.ID != commanderID {
		displayName := targetUser.Username
		if targetUser.GlobalName != "" {
			displayName = fmt.Sprintf("%s (%s)", targetUser.GlobalName, targetUser.Username)
		}
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name:    displayName,
			IconURL: targetUser.AvatarURL("128"),
		}
	}

	return embed
}

func (h *EventHandler) createCategoryButtons(prefix string, categories []string, activeCategory string, commanderID string, targetID string) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	for _, cat := range categories {
		style := discordgo.DangerButton
		if cat == activeCategory {
			style = discordgo.SuccessButton
		}

		var customID string
		if prefix == "staff" {
			customID = fmt.Sprintf("staff_page_%s", cat)
		} else {
			customID = fmt.Sprintf("%s_page_%s_%s_%s", prefix, cat, commanderID, targetID)
		}

		row = append(row, discordgo.Button{
			Label:    translateCategory(cat),
			Style:    style,
			CustomID: customID,
		})

		if len(row) == 5 {
			components = append(components, discordgo.ActionsRow{Components: row})
			row = []discordgo.MessageComponent{}
		}
	}

	if len(row) > 0 {
		components = append(components, discordgo.ActionsRow{Components: row})
	}

	if prefix != "profile" && prefix != "staff" {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Profili Görüntüle",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("profile_view_%s_%s_%s", "main", commanderID, targetID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "👤",
					},
				},
			},
		})
	} else if prefix == "profile" {

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Görev İlerlemeleri",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("tasks_page_%s_%s_%s", "milestones", commanderID, targetID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "📝",
					},
				},
			},
		})
	}

	return components
}

func (h *EventHandler) createProfileEmbed(profile *models.UserProfile, user *discordgo.User, member *discordgo.Member) *discordgo.MessageEmbed {
	level := profile.XP / 100
	remXP := profile.XP % 100

	badgeText := ""
	for _, bID := range profile.Badges {
		for _, b := range h.State.Badges {
			if b.ID == bID {
				badgeText += fmt.Sprintf("%s %s ", b.IconEmoji, b.Name)
				break
			}
		}
	}
	if badgeText == "" {
		badgeText = "Henüz rozet yok."
	}

	var description string = fmt.Sprintf("**Üyelik tarihi:** <t:%d:F>", member.JoinedAt.Unix())
	description += "\n\u200b\n"

	displayName := user.Username
	if user.GlobalName != "" {
		displayName = fmt.Sprintf("%s (%s)", user.GlobalName, user.Username)
	}

	embedFields := []*discordgo.MessageEmbedField{
		{
			Name:   "🏆 Rozetler (!badges)",
			Value:  badgeText,
			Inline: false,
		},
	}

	tempRoleText := ""
	for _, access := range profile.TempAccesses {
		if access.Type == "role" {
			tempRoleText += fmt.Sprintf("⏳ <@&%s> - <t:%d:R>\n", access.TargetID, access.ExpiresAt.Unix())
		}
	}

	if tempRoleText != "" {
		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   "⏳ Süreli Roller",
			Value:  tempRoleText,
			Inline: false,
		})
	}

	embedFields = append(embedFields, []*discordgo.MessageEmbedField{
		{
			Name:   "\u200b",
			Value:  "\u200b",
			Inline: false,
		},
		{
			Name:   "📊 Seviye",
			Value:  fmt.Sprintf("`%d`", level),
			Inline: true,
		},
		{
			Name:   "✉️ Mesajlar",
			Value:  fmt.Sprintf("`%d`", profile.MessageCount),
			Inline: true,
		},
		{
			Name:   "✨ Paylaşımlar",
			Value:  fmt.Sprintf("`%d`", profile.ShareCount),
			Inline: true,
		},
		{
			Name:   "🚀 XP İlerlemesi",
			Value:  fmt.Sprintf("%s  `%d / 100`", h.generateProgressBar(remXP, 100), remXP),
			Inline: false,
		},
	}...)

	return &discordgo.MessageEmbed{
		Title:       displayName,
		Description: description,
		Color:       0x00BCD4,
		Fields:      embedFields,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL("1024"),
		},
	}
}

func (h *EventHandler) generateProgressBar(current, max int) string {
	const barSize = 10
	if max == 0 {
		max = 1
	}
	progress := float64(current) / float64(max)
	filled := int(progress * float64(barSize))
	if filled > barSize {
		filled = barSize
	}

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "▰"
	}
	for i := filled; i < barSize; i++ {
		bar += "▱"
	}
	return bar
}

func (h *EventHandler) grantReward(s *discordgo.Session, userID, guildID string, reward models.Reward, taskID string) {
	switch reward.Type {
	case "badge":
		err := h.DB.AddBadge(userID, reward.Value)
		if err != nil {
			logger.Error("Failed to add badge %s to user %s: %v", reward.Value, userID, err)
		}
	case "role":
		err := s.GuildMemberRoleAdd(guildID, userID, reward.Value)
		if err != nil {
			logger.Error("Failed to add role %s to user %s: %v", reward.Value, userID, err)
		}
	case "xp":
		xpAmount, _ := strconv.Atoi(reward.Value)
		err := h.DB.AddXP(userID, xpAmount)
		if err != nil {
			logger.Error("Failed to add XP to user %s: %v", userID, err)
		}
		logger.Info("User %s earned %d XP from task %s", userID, xpAmount, taskID)
	case "temp_access":
		duration, err := parseDuration(reward.Duration)
		if err != nil {
			logger.Error("Failed to parse duration %s: %v", reward.Duration, err)
			return
		}

		expiresAt := time.Now().Add(duration)

		err = s.GuildMemberRoleAdd(guildID, userID, reward.Value)
		if err != nil {
			logger.Error("Failed to add temporary role %s to user %s: %v", reward.Value, userID, err)
			return
		}

		err = h.DB.AddTempAccess(userID, models.TempAccess{
			TargetID:  reward.Value,
			GuildID:   guildID,
			Type:      "role",
			ExpiresAt: expiresAt,
		})
		if err != nil {
			logger.Error("Failed to save temp access to database for %s: %v", userID, err)
		}

		logger.Info("User %s granted temporary access to role %s for %s", userID, reward.Value, reward.Duration)
	}
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

type rawModal struct {
	Type    int    `json:"type"`
	ID      string `json:"id"`
	Token   string `json:"token"`
	GuildID string `json:"guild_id"`
	Member  *struct {
		User *discordgo.User `json:"user"`
	} `json:"member"`
	Data struct {
		CustomID   string `json:"custom_id"`
		Components []struct {
			Type      int `json:"type"`
			Component *struct {
				CustomID string   `json:"custom_id"`
				Type     int      `json:"type"`
				Value    string   `json:"value"`
				Values   []string `json:"values"`
			} `json:"component"`
		} `json:"components"`
	} `json:"data"`
}

func (h *EventHandler) OnRawEvent(s *discordgo.Session, e *discordgo.Event) {
	if e.Type != "INTERACTION_CREATE" {
		return
	}

	var raw rawModal
	if err := json.Unmarshal(e.RawData, &raw); err != nil {
		logger.Error("raw unmarshal fail: %v", err)
		return
	}

	if raw.Type != 5 {
		return
	}

	if strings.HasPrefix(raw.Data.CustomID, "report_content_") {
		h.handleReportModal(s, &raw)
	} else if strings.HasPrefix(raw.Data.CustomID, "modal_staff_") {
		h.handleStaffModal(s, &raw)
	} else if strings.HasPrefix(raw.Data.CustomID, "modal_badge_") {
		h.handleBadgeModal(s, &raw)
	}
}

func (h *EventHandler) handleReportModal(s *discordgo.Session, raw *rawModal) {

	channelID := strings.Split(raw.Data.CustomID, "_")[2]
	messageID := strings.Split(raw.Data.CustomID, "_")[3]

	msg, err := s.ChannelMessage(channelID, messageID)
	if err != nil {
		h.respondRaw(s, raw.ID, raw.Token, "❌ İçerik bulunamadı, tekrar deneyin.")
		return
	}

	var reportReason string
	var isEmergency bool

	for _, comp := range raw.Data.Components {
		if comp.Component == nil {
			continue
		}
		switch comp.Component.CustomID {
		case "report_content":
			reportReason = comp.Component.Value
		case "emergency_checkbox":
			if len(comp.Component.Values) > 0 && comp.Component.Values[0] == "yes" {
				isEmergency = true
			}
		}
	}

	if h.Config.ReportLogChannel == "" {
		h.respondRaw(s, raw.ID, raw.Token, "❌ Rapor log kanalı ayarlanmamış, lütfen yetkililere bildirin.")
		return
	}

	reporter := raw.Member.User
	var reporterStr string
	if reporter != nil {
		reporterStr = fmt.Sprintf("<@%s> (%s)", reporter.ID, reporter.Username)
	} else {
		reporterStr = "Bilinmeyen Kullanıcı"
	}

	jumpURL := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", raw.GuildID, channelID, messageID)

	color := 0xFFA500
	title := "📋 Yeni İçerik Raporu"
	content := ""

	if isEmergency {
		color = 0xFF0000
		title = "🚨 ACİL İçerik Raporu"
		content = "@everyone"
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: fmt.Sprintf("**Şikayet Nedeni:**\n%s", reportReason),
		Color:       color,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Raporlayan",
				Value:  reporterStr,
				Inline: true,
			},
			{
				Name:   "Mesaj Sahibi",
				Value:  fmt.Sprintf("<@%s> (%s)", msg.Author.ID, msg.Author.Username),
				Inline: true,
			},
			{
				Name:   "Mesaj Tarihi",
				Value:  fmt.Sprintf("<t:%d:R>", msg.Timestamp.Unix()),
				Inline: true,
			},
			{
				Name:   "İçerik Bağlantısı",
				Value:  fmt.Sprintf("[Mesaja Git](%s)", jumpURL),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Mesaj ID: %s", messageID),
		},
	}

	if msg.Author.Avatar != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: "https://cdn3.emoji.gg/emojis/4645-report-message.png",
		}
	}

	_, err = s.ChannelMessageSendComplex(h.Config.ReportLogChannel, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})

	if err != nil {
		logger.Error("Failed to send report log: %v", err)
		h.respondRaw(s, raw.ID, raw.Token, "❌ Rapor iletilirken bir hata oluştu.")
		return
	}

	h.respondRaw(s, raw.ID, raw.Token, "✅ Raporunuz başarıyla yönetime iletildi. İlginiz için teşekkürler <3")
}

func (h *EventHandler) handleStaffModal(s *discordgo.Session, raw *rawModal) {

	action := strings.Split(raw.Data.CustomID, "_")[2]

	var selectedUserIDs []string
	var selectedCategory string
	var send_dm bool = false

	for _, comp := range raw.Data.Components {
		if comp.Component == nil {
			continue
		}
		switch comp.Component.CustomID {
		case "user_selected":
			selectedUserIDs = comp.Component.Values
		case "class_radio":
			selectedCategory = comp.Component.Value
		case "send_dm":
			for _, value := range comp.Component.Values {
				if value == "yes" {
					send_dm = true
				}
			}
		}
	}

	if len(selectedUserIDs) == 0 {
		h.respondRaw(s, raw.ID, raw.Token, "❌ Kullanıcı seçilmedi.")
		return
	}

	roleID, ok := h.Config.StaffRoles[selectedCategory]
	if !ok && action == "add" {
		h.respondRaw(s, raw.ID, raw.Token, fmt.Sprintf("❌ Geçersiz kategori: `%s`", selectedCategory))
		return
	}

	var msgs []string
	for _, userID := range selectedUserIDs {
		var err error
		if action == "add" {
			for _, rID := range h.Config.StaffRoles {
				s.GuildMemberRoleRemove(raw.GuildID, userID, rID)
			}

			err = s.GuildMemberRoleAdd(raw.GuildID, userID, roleID)
			if err == nil {
				msgs = append(msgs, fmt.Sprintf("✅ <@%s> → `%s` rolü verildi.", userID, selectedCategory))
			}
		} else {
			var removedRoles []string

			member, err := s.GuildMember(raw.GuildID, userID)

			if err != nil {
				continue
			}

			var memberRoles []string = member.Roles

			for _, i := range h.Config.StaffRoles {
				for j, x := range memberRoles {
					if i == x {
						err = s.GuildMemberRoleRemove(raw.GuildID, userID, x)
						if err == nil {
							removedRoles = append(removedRoles, "<@&"+x+">")
							memberRoles = append(memberRoles[:j], memberRoles[j+1:]...)
							break
						}
					}
				}
			}

			if len(removedRoles) == 0 {
				msgs = append(msgs, fmt.Sprintf("❌ <@%s> → Herhangi bir yetkisi bulunamadı.", userID))
				break
			}

			var text string

			if len(removedRoles) > 1 {
				text = "rolleri"
			} else {
				text = "rolü"
			}

			msgs = append(msgs, fmt.Sprintf("✅ <@%s> → %s %s alındı.", userID, strings.Join(removedRoles, ", "), text))
		}

		if err != nil {
			msgs = append(msgs, fmt.Sprintf("❌ <@%s> için işlem başarısız: %v", userID, err))
		}
	}

	h.respondRaw(s, raw.ID, raw.Token, strings.Join(msgs, "\n"))

	if send_dm {
		for _, userID := range selectedUserIDs {
			channel, err := s.UserChannelCreate(userID)
			if err != nil {
				continue
			}
			_, err = s.ChannelMessageSend(channel.ID, "**⚠️ Bilgilendirme:** Bir yönetici tarafından tüm yetkileriniz alındı. Sunucumuzda artık normal üye olarak devam edeceksiniz.\n\nEğer bir hata olduğunu düşünüyorsanız yetkili birisi ile iletişime geçiniz.")
			if err != nil {
				continue
			}
		}
	}
}

func (h *EventHandler) handleBadgeModal(s *discordgo.Session, raw *rawModal) {
	action := strings.Split(raw.Data.CustomID, "_")[2]

	var selectedUserIDs []string
	var selectedBadge string

	for _, comp := range raw.Data.Components {
		if comp.Component == nil {
			continue
		}
		switch comp.Component.CustomID {
		case "user_selected":
			selectedUserIDs = comp.Component.Values
		case "badge_selected":
			if len(comp.Component.Values) > 0 {
				selectedBadge = comp.Component.Values[0]
			}
		}
	}

	if len(selectedUserIDs) == 0 {
		h.respondRaw(s, raw.ID, raw.Token, "❌ Kullanıcı seçilmedi.")
		return
	}

	if selectedBadge == "" {
		h.respondRaw(s, raw.ID, raw.Token, "❌ Rozet seçilmedi.")
		return
	}

	var badgeName string
	var badgeIcon string

	for _, badge := range h.State.Badges {
		if badge.ID == selectedBadge {
			badgeName = badge.Name
			badgeIcon = badge.IconEmoji
			break
		}
	}

	var msgs []string
	for _, userID := range selectedUserIDs {
		var err error
		switch action {
		case "add":
			err = h.DB.AddBadge(userID, selectedBadge)
			if err == nil {
				msgs = append(msgs, fmt.Sprintf("✅ <@%s> → **%s %s** rozeti verildi.", userID, badgeIcon, badgeName))
			} else {
				msgs = append(msgs, fmt.Sprintf("❌ <@%s> rozet verilemedi: %v", userID, err))
			}
		case "remove":
			err = h.DB.RemoveBadge(userID, selectedBadge)
			if err == nil {
				msgs = append(msgs, fmt.Sprintf("✅ <@%s> → **%s %s** rozeti alındı.", userID, badgeIcon, badgeName))
			} else {
				msgs = append(msgs, fmt.Sprintf("❌ <@%s> rozet alınamadı: %v", userID, err))
			}
		}
	}

	h.respondRaw(s, raw.ID, raw.Token, strings.Join(msgs, "\n"))
}

func (h *EventHandler) respondRaw(s *discordgo.Session, interactionID, token, content string) {
	endpoint := discordgo.EndpointInteractionResponse(interactionID, token)
	payload := map[string]interface{}{
		"type": 4,
		"data": map[string]interface{}{
			"content": content,
			"flags":   64,
		},
	}
	_, err := s.RequestWithBucketID("POST", endpoint, payload, endpoint)
	if err != nil {
		logger.Error("Failed to respond to raw interaction: %v", err)
	}
}

func (h *EventHandler) createStaffEmbed(s *discordgo.Session, guildID, category string) *discordgo.MessageEmbed {
	roleID := h.Config.StaffRoles[category]
	if roleID == "" {
		return &discordgo.MessageEmbed{
			Title:       "Hata",
			Description: "Bu kategori için rol yapılandırması bulunamadı.",
			Color:       0xFF0000,
		}
	}

	members, err := s.GuildMembers(guildID, "", 1000)
	if err != nil {
		return &discordgo.MessageEmbed{
			Title:       "Hata",
			Description: "Üye listesi alınamadı.",
			Color:       0xFF0000,
		}
	}

	var staffList []string
	for _, m := range members {
		for _, r := range m.Roles {
			if r == roleID {
				name := m.User.Username
				if m.Nick != "" {
					name = fmt.Sprintf("%s (%s)", m.Nick, m.User.Username)
				}
				staffList = append(staffList, fmt.Sprintf("• <@%s> - **%s**", m.User.ID, name))
				break
			}
		}
	}

	description := strings.Join(staffList, "\n")
	if len(staffList) == 0 {
		description = "*Bu kategoride henüz yetkili bulunmuyor.*"
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("👥 Yetkili Listesi - %s", translateCategory(category)),
		Description: description,
		Color:       0x2196F3,
	}
}
