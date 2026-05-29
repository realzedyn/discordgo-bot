package application

import (
	"discord-bot/internal/commands"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type StaffCommand struct{}

func (c *StaffCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "yetkili",
		Description: "Yetkili yönetimi ve listeleme",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "listele",
				Description: "Yetkilileri kategorilere göre listeler",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "ekle",
				Description: "Yeni bir yetkili ekler",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "çıkar",
				Description: "Bir yetkiliyi görevden çıkarır",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}
}

func (c *StaffCommand) Execute(ctx *commands.ApplicationContext) error {
	data := ctx.Interaction.ApplicationCommandData()
	subcommand := data.Options[0].Name

	switch subcommand {
	case "listele":
		return c.handleList(ctx)
	case "ekle":
		return c.handleAdd(ctx)
	case "çıkar":
		return c.handleRemove(ctx)
	}

	return nil
}

func (c *StaffCommand) handleList(ctx *commands.ApplicationContext) error {
	category := "admin"
	roleID := ctx.Config.StaffRoles[category]

	var description string
	if roleID == "" {
		description = "Bu kategori için rol yapılandırması bulunamadı."
	} else {
		members, _ := ctx.Session.GuildMembers(ctx.Interaction.GuildID, "", 1000)
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
		description = strings.Join(staffList, "\n")
		if len(staffList) == 0 {
			description = "*Bu kategoride henüz yetkili bulunmuyor.*"
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("👥 Yetkili Listesi - %s", "Yönetici"),
		Description: description,
		Color:       0x2196F3,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Sunucu yönetim ekibi.",
		},
	}

	categoryOrder := []string{"admin", "moderator", "trial_moderator", "sharer"}
	var components []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	categoryTranslations := map[string]string{
		"admin":           "Yönetici",
		"moderator":       "Moderatör",
		"trial_moderator": "Deneme Moderatör",
		"sharer":          "Paylaşımcı",
	}

	for _, cat := range categoryOrder {
		style := discordgo.DangerButton
		if cat == category {
			style = discordgo.SuccessButton
		}

		row = append(row, discordgo.Button{
			Label:    categoryTranslations[cat],
			Style:    style,
			CustomID: fmt.Sprintf("staff_page_%s", cat),
		})
	}
	components = append(components, discordgo.ActionsRow{Components: row})

	return ctx.Session.InteractionRespond(ctx.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func (c *StaffCommand) handleAdd(ctx *commands.ApplicationContext) error {

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

	data := map[string]interface{}{
		"type": 9,
		"data": map[string]interface{}{
			"custom_id": "modal_staff_add_" + ctx.Interaction.Member.User.ID,
			"title":     "Yetkili Ekle / Güncelle",
			"components": []map[string]interface{}{
				{
					"type":    10,
					"content": "### Bilgilendirme:\n> Yetki güncelleme işlemi de burdan yapılmaktadır.",
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
					"type":        18,
					"label":       "Yetki seçin",
					"description": "Kullanıcının bütün yetki rolleri alınıp sadece seçtiğiniz yetki yazılacaktır.",
					"component": map[string]interface{}{
						"type":      21,
						"custom_id": "class_radio",
						"options": []map[string]interface{}{
							{"value": "admin", "label": "Admin", "description": "Büyük patron, her şeyi görür ve yönetir."},
							{"value": "moderator", "label": "Moderator", "description": "Düzeni sağlar, ban çekicini elinden düşürmez."},
							{"value": "trial_moderator", "label": "Deneme Moderator", "description": "Henüz stajyer, yetki almak için ter döküyor."},
							{"value": "sharer", "label": "Paylaşımcı", "description": "İçerik makinesi, her şeyi ilk o sızdırır."},
						},
					},
				},
			},
		},
	}

	endpoint := discordgo.EndpointInteractionResponse(ctx.Interaction.Interaction.ID, ctx.Interaction.Interaction.Token)
	_, err := ctx.Session.RequestWithBucketID("POST", endpoint, data, endpoint)
	return err
}

func (c *StaffCommand) handleRemove(ctx *commands.ApplicationContext) error {

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

	data := map[string]interface{}{
		"type": 9,
		"data": map[string]interface{}{
			"custom_id": "modal_staff_remove_" + ctx.Interaction.Member.User.ID,
			"title":     "Yetkili Çıkar (Kov)",
			"components": []map[string]interface{}{
				{
					"type":    10,
					"content": "### Bilgilendirme:\n> Seçtiğiniz kullanıcının bütün yetkili rolleri silinecek olup tüm ilişiği kesilecektir.",
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
					"label": "Kullanıcıyı bilgilendir",
					"component": map[string]interface{}{
						"type":      22,
						"custom_id": "send_dm",
						"required":  false,
						"options": []map[string]interface{}{
							{"value": "yes", "label": "DM Gönder"},
						},
					},
				},
			},
		},
	}

	endpoint := discordgo.EndpointInteractionResponse(ctx.Interaction.Interaction.ID, ctx.Interaction.Interaction.Token)
	_, err := ctx.Session.RequestWithBucketID("POST", endpoint, data, endpoint)
	return err
}
