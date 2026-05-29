package application

import (
	"discord-bot/internal/commands"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type ReportContextCommand struct{}

func (c *ReportContextCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name: "İçeriği Raporla",
		Type: discordgo.MessageApplicationCommand,
	}
}

func (c *ReportContextCommand) Execute(ctx *commands.ApplicationContext) error {
	data := ctx.Interaction.ApplicationCommandData()
	msg := data.Resolved.Messages[data.TargetID]

	if msg == nil {
		return fmt.Errorf("target message not found")
	}

	raw_modal := map[string]interface{}{
		"type": 9,
		"data": map[string]interface{}{
			"custom_id": "report_content_" + msg.ChannelID + "_" + data.TargetID,
			"title":     "Rapor Formu",
			"components": []map[string]interface{}{
				{
					"type":    10,
					"content": "### Raporlanan İçeriğin Detayları:\n" + fmt.Sprintf("> Mesaj ID: **%s** (%s kanalında)\n> Gönderen: **%s** (%s)", data.TargetID, fmt.Sprintf("<#%s>", ctx.Interaction.ChannelID), msg.Author.Username, fmt.Sprintf("<t:%d:R>", msg.Timestamp.Unix())),
				},
				{
					"type":        18,
					"label":       "Şikayet nedeni",
					"description": "Lütfen detaylı şekilde bu içeriği neden raporladığınızı belirtiniz.",
					"component": map[string]interface{}{
						"type":        4,
						"custom_id":   "report_content",
						"style":       2,
						"min_length":  20,
						"max_length":  4000,
						"placeholder": "Raporunuzu buraya yazınız...",
						"required":    true,
					},
				},
				{
					"type":        18,
					"label":       "Aciliyet durumu",
					"description": "Eğer gerçekten acil bir durum olduğunu kutucuğu işaretleyin.",
					"component": map[string]interface{}{
						"type":      22,
						"custom_id": "emergency_checkbox",
						"options": []map[string]interface{}{
							{"value": "yes", "label": "Acil Müdahale Gerekiyor"},
						},
						"required": false,
					},
				},
			},
		},
	}

	endpoint := discordgo.EndpointInteractionResponse(ctx.Interaction.Interaction.ID, ctx.Interaction.Interaction.Token)
	_, err := ctx.Session.RequestWithBucketID("POST", endpoint, raw_modal, endpoint)
	return err
}
