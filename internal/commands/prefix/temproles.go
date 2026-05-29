package prefix

import (
	"discord-bot/internal/commands"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type TempRolesCommand struct{}

func (c *TempRolesCommand) Name() string {
	return "temproles"
}

func (c *TempRolesCommand) Aliases() []string {
	return []string{"geçiciroller", "temprole"}
}

func (c *TempRolesCommand) Description() string {
	return "Aktif geçici rolleri listeler (Sadece Admin)."
}

func (c *TempRolesCommand) Execute(ctx *commands.Context) error {

	isAdmin := false
	for _, adminID := range ctx.Config.Admins {
		if ctx.Message.Author.ID == adminID {
			isAdmin = true
			break
		}
	}

	for _, roleID := range ctx.Message.Member.Roles {
		if roleID == ctx.Config.StaffRoles["admin"] {
			isAdmin = true
			break
		}
	}

	if !isAdmin {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "❌ Bu komutu sadece yöneticiler kullanabilir.", ctx.Message.Reference())
		return err
	}

	profiles, err := ctx.DB.GetAllProfiles()
	if err != nil {
		return err
	}

	var builder strings.Builder
	count := 0

	for _, profile := range profiles {
		for _, access := range profile.TempAccesses {
			if access.Type == "role" {
				count++
				builder.WriteString(fmt.Sprintf("👤 <@%s> - <@&%s>\n", profile.UserID, access.TargetID))
				builder.WriteString(fmt.Sprintf("└ Bitiş: <t:%d:R> (<t:%d:F>)\n\n", access.ExpiresAt.Unix(), access.ExpiresAt.Unix()))
			}
		}
	}

	if count == 0 {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "ℹ️ Şu an aktif geçici rol bulunmuyor.", ctx.Message.Reference())
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⏳ Aktif Geçici Roller",
		Description: builder.String(),
		Color:       0x00BCD4,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Toplam %d aktif kayıt.", count),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
