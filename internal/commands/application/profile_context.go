package application

import (
	"discord-bot/internal/commands"

	"github.com/bwmarrin/discordgo"
)

type ProfileContextCommand struct{}

func (c *ProfileContextCommand) Definition() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name: "Profilini Gör",
		Type: discordgo.UserApplicationCommand,
	}
}

func (c *ProfileContextCommand) Execute(ctx *commands.ApplicationContext) error {
	data := ctx.Interaction.ApplicationCommandData()

	targetMember, err := ctx.Session.GuildMember(
		ctx.Interaction.GuildID,
		data.TargetID,
	)

	if err != nil {
		return ctx.Session.InteractionRespond(ctx.Interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ " + err.Error(),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	targetUser := targetMember.User

	profile, err := ctx.DB.GetProfile(targetUser.ID)
	if err != nil {
		return err
	}

	var authorID string
	if ctx.Interaction.Member != nil {
		authorID = ctx.Interaction.Member.User.ID
	} else if ctx.Interaction.User != nil {
		authorID = ctx.Interaction.User.ID
	}

	embed, components := commands.BuildProfileEmbed(targetUser, targetMember, profile, ctx.State.Badges, authorID)

	err = ctx.Session.InteractionRespond(ctx.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
	return err
}
