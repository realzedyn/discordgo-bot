package prefix

import (
	"discord-bot/internal/commands"
	"fmt"
)

type PingCommand struct{}

func (c *PingCommand) Name() string {
	return "ping"
}

func (c *PingCommand) Aliases() []string {
	return []string{"p", "latency"}
}

func (c *PingCommand) Description() string {
	return "Measure the bot's latency."
}

func (c *PingCommand) Execute(ctx *commands.Context) error {
	_, err := ctx.Session.ChannelMessageSendReply(ctx.Message.ChannelID, fmt.Sprintf("Pong! 🏓 (Latency: %vms)", ctx.Session.HeartbeatLatency().Milliseconds()), ctx.Message.Reference())
	return err
}
