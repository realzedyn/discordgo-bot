package application

import (
	"discord-bot/internal/commands"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type BadgeCommand struct{}

func (c *BadgeCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "rozet",
		Description: "Rozet yönetimi",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "ekle",
				Description: "Bir kullanıcıya rozet ekler",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "çıkar",
				Description: "Bir kullanıcıdan rozet çıkarır",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *BadgeCommand) Execute(ctx *commands.ApplicationContext) error {
	data := ctx.Interaction.ApplicationCommandData()
	subcommand := data.Options[0].Name

	switch subcommand {
	case "ekle":
		return c.handleAdd(ctx)
	case "çıkar":
		return c.handleRemove(ctx)
	}

	return nil
}

func (c *BadgeCommand) getBadgeOptions(ctx *commands.ApplicationContext) []map[string]interface{} {
	var options []map[string]interface{}

	categoryTranslations := map[string]string{
		"milestones":   "Ana Görevler",
		"daily":        "Günlük Görevler",
		"staff":        "Yetkili",
		"achievements": "Başarımlar",
		"special":      "Özel",
	}

	for i, badge := range ctx.State.Badges {
		if i >= 25 {
			break
		}

		catName, ok := categoryTranslations[badge.Category]
		if !ok {
			catName = badge.Category
		}

		desc := fmt.Sprintf("[%s] - %s", catName, badge.Description)
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}

		options = append(options, map[string]interface{}{
			"label":       badge.Name,
			"value":       badge.ID,
			"description": desc,
			"emoji": map[string]interface{}{
				"name": badge.IconEmoji,
			},
		})
	}

	return options
}

func (c *BadgeCommand) handleAdd(ctx *commands.ApplicationContext) error {
	isAdmin := false
	for _, adminID := range ctx.Config.Admins {
		if ctx.Interaction.Member.User.ID == adminID {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return ctx.Session.InteractionRespond(ctx.Interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Bu komutu sadece bot yöneticileri kullanabilir.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := c.getBadgeOptions(ctx)

	data := map[string]interface{}{
		"type": 9,
		"data": map[string]interface{}{
			"custom_id": "modal_badge_add_" + ctx.Interaction.Member.User.ID,
			"title":     "Rozet Ekle",
			"components": []map[string]interface{}{
				{
					"type":    10,
					"content": "### Bilgilendirme:\n> Rozet sistemi aktiviteyi arttırmak için tasarlanmıştır. Gerekmeyen durumlarda manuel olarak kullanıcılara verilmesi tavsiye edilmez.",
				},
				{
					"type":  18,
					"label": "İşlem yapılacak kullanıcıyı seçiniz.",
					"component": map[string]interface{}{
						"type":       5,
						"custom_id":  "user_selected",
						"max_values": 5,
						"required":   true,
					},
				},
				{
					"type":  18,
					"label": "Rozet seçiniz.",
					"component": map[string]interface{}{
						"type":        3,
						"custom_id":   "badge_selected",
						"placeholder": "Eklenecek rozeti seçin...",
						"options":     options,
					},
				},
			},
		},
	}

	endpoint := discordgo.EndpointInteractionResponse(ctx.Interaction.Interaction.ID, ctx.Interaction.Interaction.Token)
	_, err := ctx.Session.RequestWithBucketID("POST", endpoint, data, endpoint)
	return err
}

func (c *BadgeCommand) handleRemove(ctx *commands.ApplicationContext) error {
	isAdmin := false
	for _, adminID := range ctx.Config.Admins {
		if ctx.Interaction.Member.User.ID == adminID {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		return ctx.Session.InteractionRespond(ctx.Interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Bu komutu sadece bot yöneticileri kullanabilir.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := c.getBadgeOptions(ctx)

	data := map[string]interface{}{
		"type": 9,
		"data": map[string]interface{}{
			"custom_id": "modal_badge_remove_" + ctx.Interaction.Member.User.ID,
			"title":     "Rozet Çıkar",
			"components": []map[string]interface{}{
				{
					"type":    10,
					"content": "### Bilgilendirme:\n> Gerekmeyen durumlarda manuel olarak kullanıcılardan rozet alınması tavsiye edilmez.",
				},
				{
					"type":  18,
					"label": "İşlem yapılacak kullanıcıyı seçiniz.",
					"component": map[string]interface{}{
						"type":       5,
						"custom_id":  "user_selected",
						"max_values": 5,
						"required":   true,
					},
				},
				{
					"type":  18,
					"label": "Rozet seçiniz.",
					"component": map[string]interface{}{
						"type":        3,
						"custom_id":   "badge_selected",
						"placeholder": "Çıkarılacak rozeti seçin...",
						"options":     options,
					},
				},
			},
		},
	}

	endpoint := discordgo.EndpointInteractionResponse(ctx.Interaction.Interaction.ID, ctx.Interaction.Interaction.Token)
	_, err := ctx.Session.RequestWithBucketID("POST", endpoint, data, endpoint)
	return err
}
