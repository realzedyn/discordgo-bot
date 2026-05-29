package prefix

import (
	"discord-bot/internal/commands"

	"github.com/bwmarrin/discordgo"
)

type ProfileCommand struct{}

func (c *ProfileCommand) Name() string {
	return "profile"
}

func (c *ProfileCommand) Aliases() []string {
	return []string{"p", "profil"}
}

func (c *ProfileCommand) Description() string {
	return "Kullanıcı profilini ve istatistiklerini gösterir."
}

func (c *ProfileCommand) Execute(ctx *commands.Context) error {
	targetUser, targetMember, err := resolveTargetUser(ctx)
	if err != nil {
		_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, "❌ "+err.Error(), ctx.Message.Reference())
		return err
	}

	profile, err := ctx.DB.GetProfile(targetUser.ID)
	if err != nil {
		return err
	}

	embed, components := commands.BuildProfileEmbed(targetUser, targetMember, profile, ctx.State.Badges, ctx.Message.Author.ID)

	_, err = ctx.Session.ChannelMessageSendComplex(ctx.Message.ChannelID, &discordgo.MessageSend{
		Embed:      embed,
		Components: components,
		Reference:  ctx.Message.Reference(),
	})
	return err
}
