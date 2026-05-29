package prefix

import (
	"discord-bot/internal/commands"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct {
	Registry *commands.Registry
}

func (c *HelpCommand) Name() string {
	return "yardım"
}

func (c *HelpCommand) Aliases() []string {
	return []string{"help", "y", "komutlar"}
}

func (c *HelpCommand) Description() string {
	return "Botun tüm komutlarını ve ne işe yaradıklarını listeler."
}

func (c *HelpCommand) Execute(ctx *commands.Context) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Bot Komut Listesi",
		Description: fmt.Sprintf("Aşağıda botun kullanabileceğiniz tüm komutları yer almaktadır. Komut prefixi: `%s`", ctx.Config.Prefix),
		Color:       0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: ctx.Session.State.User.AvatarURL("256"),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Daha fazla bilgi için komutları inceleyin.",
		},
	}

	uniqueCommands := make(map[string]commands.Command)
	for _, cmd := range c.Registry.Commands {
		uniqueCommands[cmd.Name()] = cmd
	}

	var cmdNames []string
	for name := range uniqueCommands {
		cmdNames = append(cmdNames, name)
	}
	sort.Strings(cmdNames)

	for _, name := range cmdNames {
		cmd := uniqueCommands[name]

		if cmd.Name() == "yardım" {
			continue
		}

		var aliasesStr string
		if len(cmd.Aliases()) > 0 {
			aliasesStr = fmt.Sprintf("\n*Alternatifler: `%s`*", strings.Join(cmd.Aliases(), "`, `"))
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("🔹 %s%s", ctx.Config.Prefix, cmd.Name()),
			Value:  fmt.Sprintf("%s%s", cmd.Description(), aliasesStr),
			Inline: false,
		})
	}

	_, err := ctx.Session.ChannelMessageSendEmbedReply(ctx.Message.ChannelID, embed, ctx.Message.Reference())
	return err
}
