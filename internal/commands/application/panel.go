package application

import (
	"discord-bot/internal/commands"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PanelCommand struct{}

func (c *PanelCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "panel",
		Description: "Hazır panelleri gönderir.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "tip",
				Description: "Gönderilecek paneli seçin.",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Sharer",
						Value: "sharer",
					},
					{
						Name:  "Ticket",
						Value: "ticket",
					},
				},
			},
		},
	}
}

func (c *PanelCommand) Execute(ctx *commands.ApplicationContext) error {
	data := ctx.Interaction.ApplicationCommandData()

	if len(data.Options) > 0 {
		selectedVal := data.Options[0].StringValue()
		if selectedVal == "sharer" {
			c.handleSharer(ctx)

		}

		if selectedVal == "ticket" {
			c.handleTicket(ctx)
		}

		return ctx.Session.InteractionRespond(ctx.Interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: fmt.Sprintf("✅ `%s` paneli başarıyla gönderildi.", strings.ToUpper(selectedVal)),
			},
		})
	}

	return nil
}

func (c *PanelCommand) handleSharer(ctx *commands.ApplicationContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "📢 Paylaşımcı başvurusu",
		Description: "Aşağıdaki butona tıklayarak sunucumuzda **Paylaşımcı** rolü için başvuruda bulunabilirsiniz!\n\n*Not: Başvuru öncesinde kuralları okuduğunuzdan emin olun.*",
		Color:       0x9B59B6,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Paylaşımcı başvurusu şu an kullanıma hazır değildir. Geliştirilme aşamasında...",
		},
	}

	button := discordgo.Button{
		Label:    "Başvur",
		Style:    discordgo.PrimaryButton,
		CustomID: "btn_sharer_apply",
		Emoji: &discordgo.ComponentEmoji{
			Name: "📝",
		},
		Disabled: true,
	}

	ctx.Session.ChannelMessageSendComplex(ctx.Interaction.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{button},
			},
		},
	})

	return nil
}

func (c *PanelCommand) handleTicket(ctx *commands.ApplicationContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "📩 Destek talebi oluştur",
		Description: "Herhangi bir konuda yardım almak için aşağıdaki butona tıklayın ve karşınıza çıkan formu doldurun.",
		Color:       0x9B59B6,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Destek sistemi şu an kullanıma hazır değildir. Geliştirilme aşamasında...",
		},
	}

	button := discordgo.Button{
		Label:    "Destek Al",
		Style:    discordgo.PrimaryButton,
		CustomID: "btn_ticket_apply",
		Emoji: &discordgo.ComponentEmoji{
			Name: "📩",
		},
		Disabled: true,
	}

	ctx.Session.ChannelMessageSendComplex(ctx.Interaction.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{button},
			},
		},
	})
	return nil
}
