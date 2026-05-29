package prefix

import (
	"discord-bot/internal/commands"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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

func createCategoryButtons(prefix string, categories []string, activeCategory string, commanderID string, targetID string) []discordgo.MessageComponent {
	var components []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	for _, cat := range categories {
		style := discordgo.DangerButton
		if cat == activeCategory {
			style = discordgo.SuccessButton
		}

		row = append(row, discordgo.Button{
			Label:    translateCategory(cat),
			Style:    style,
			CustomID: fmt.Sprintf("%s_page_%s_%s_%s", prefix, cat, commanderID, targetID),
		})

		if len(row) == 5 {
			components = append(components, discordgo.ActionsRow{Components: row})
			row = []discordgo.MessageComponent{}
		}
	}

	if len(row) > 0 {
		components = append(components, discordgo.ActionsRow{Components: row})
	}

	if prefix != "profile" {
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
	}

	return components
}
func resolveTargetUser(ctx *commands.Context) (*discordgo.User, *discordgo.Member, error) {
	var targetUser *discordgo.User
	var targetMember *discordgo.Member

	if len(ctx.Args) == 0 {
		targetUser = ctx.Message.Author
		targetMember = ctx.Message.Member
	} else {

		if len(ctx.Message.Mentions) > 0 {
			targetUser = ctx.Message.Mentions[0]
			targetMember, _ = ctx.Session.GuildMember(ctx.Message.GuildID, targetUser.ID)
		} else {

			input := ctx.Args[0]
			member, err := ctx.Session.GuildMember(ctx.Message.GuildID, input)
			if err == nil {
				targetMember = member
				targetUser = member.User
			} else {

				members, err := ctx.Session.GuildMembersSearch(ctx.Message.GuildID, input, 1)
				if err == nil && len(members) > 0 {
					targetMember = members[0]
					targetUser = targetMember.User
				}
			}
		}
	}

	if targetUser == nil || targetMember == nil {
		return nil, nil, fmt.Errorf("Kullanıcı bulunamadı")
	}
	return targetUser, targetMember, nil
}
