package commands

import (
	"discord-bot/internal/models"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func GenerateProgressBar(current, max int) string {
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

func BuildProfileEmbed(targetUser *discordgo.User, targetMember *discordgo.Member, profile *models.UserProfile, stateBadges []models.Badge, authorID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level := profile.XP / 100
	remXP := profile.XP % 100

	badgeText := ""
	for _, bID := range profile.Badges {
		for _, b := range stateBadges {
			if b.ID == bID {
				badgeText += fmt.Sprintf("%s %s ", b.IconEmoji, b.Name)
				break
			}
		}
	}
	if badgeText == "" {
		badgeText = "Henüz rozet yok."
	}

	var description string
	if targetMember != nil {
		description = fmt.Sprintf("**Üyelik tarihi:** <t:%d:F>", targetMember.JoinedAt.Unix())
	}

	if targetUser.ID != authorID {
		if description != "" {
			description += "\n\n"
		}
		description += fmt.Sprintf("**Son görülme:** <t:%d:F>", profile.LastMessageAt.Unix())
	}

	description += "\n\u200b\n"

	displayName := targetUser.Username
	if targetUser.GlobalName != "" {
		displayName = fmt.Sprintf("%s (%s)", targetUser.GlobalName, targetUser.Username)
	}

	embedFields := []*discordgo.MessageEmbedField{
		{
			Name:   "🏆 Rozetler - !badges",
			Value:  badgeText,
			Inline: false,
		},
	}

	tempRoleText := ""
	for _, access := range profile.TempAccesses {
		if access.Type == "role" {
			tempRoleText += fmt.Sprintf("- <@&%s> - <t:%d:R> süresi dolacak.\n", access.TargetID, access.ExpiresAt.Unix())
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
			Value:  fmt.Sprintf("%s  `%d / 100`", GenerateProgressBar(remXP, 100), remXP),
			Inline: false,
		},
	}...)

	embed := &discordgo.MessageEmbed{
		Title:       displayName,
		Description: description,
		Color:       0x00BCD4,
		Fields:      embedFields,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: targetUser.AvatarURL("1024"),
		},
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Görev İlerlemeleri",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("%s_page_%s_%s_%s", "tasks", "milestones", authorID, targetUser.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "📝",
					},
				},
			},
		},
	}

	return embed, components
}
