package prefix

import (
	"discord-bot/internal/commands"
	"discord-bot/internal/models"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type BadgesCommand struct{}

func (c *BadgesCommand) Name() string {
	return "badges"
}

func (c *BadgesCommand) Aliases() []string {
	return []string{"rozetler", "rozet", "badge"}
}

func (c *BadgesCommand) Description() string {
	return "Sistemdeki tüm rozetleri listeler."
}

func (c *BadgesCommand) Execute(ctx *commands.Context) error {
	targetUser, _, err := resolveTargetUser(ctx)
	if err != nil {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "❌ "+err.Error(), ctx.Message.Reference())
		return err
	}

	profile, err := ctx.DB.GetProfile(targetUser.ID)
	if err != nil {
		return err
	}

	categories := make(map[string][]models.Badge)
	var categoryOrder []string

	for _, badge := range ctx.State.Badges {
		if _, ok := categories[badge.Category]; !ok {
			categoryOrder = append(categoryOrder, badge.Category)
		}
		categories[badge.Category] = append(categories[badge.Category], badge)
	}

	if len(categoryOrder) == 0 {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "❌ Hiç rozet tanımlanmamış.", ctx.Message.Reference())
		return err
	}

	firstCategory := categoryOrder[0]
	embed := createBadgesEmbed(profile, firstCategory, categories[firstCategory], targetUser, ctx)
	components := createCategoryButtons("badges", categoryOrder, firstCategory, ctx.Message.Author.ID, targetUser.ID)

	_, err = ctx.Session.ChannelMessageSendComplex(ctx.Message.ChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
		Reference:  ctx.Message.Reference(),
	})

	return err
}

func createBadgesEmbed(profile *models.UserProfile, category string, badges []models.Badge, targetUser *discordgo.User, ctx *commands.Context) *discordgo.MessageEmbed {
	var builder strings.Builder

	for _, badge := range badges {
		status := "🔒"

		for _, bID := range profile.Badges {
			if bID == badge.ID {
				status = "🔓"
				break
			}
		}

		builder.WriteString(fmt.Sprintf("%s %s **%s**\n", status, badge.IconEmoji, badge.Name))
		builder.WriteString(fmt.Sprintf("└ *%s*\n\n", badge.Description))
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🏆 Rozetler - %s", translateCategory(category)),
		Description: builder.String(),
		Color:       0xFFD700,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Bu rozetler sadece yetkililere tanımlanır.",
		},
	}

	if targetUser.ID != ctx.Message.Author.ID {
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
