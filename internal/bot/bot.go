package bot

import (
	"discord-bot/internal/commands"
	"discord-bot/internal/commands/application"
	"discord-bot/internal/commands/prefix"
	"discord-bot/internal/config"
	"discord-bot/internal/database"
	"discord-bot/internal/events"
	"discord-bot/internal/logger"
	"discord-bot/internal/state"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session     *discordgo.Session
	Config      *config.Config
	Registry    *commands.Registry
	AppRegistry *commands.ApplicationRegistry
	DB          *database.Database
	State       *state.Manager
}

func New(cfg *config.Config) *Bot {
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		logger.Error("Failed to create Discord session: %v", err)
		return nil
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	registry := commands.NewRegistry(cfg.Prefix)

	registry.Register(&prefix.PingCommand{})
	registry.Register(&prefix.ProfileCommand{})
	registry.Register(&prefix.TasksCommand{})
	registry.Register(&prefix.BadgesCommand{})
	registry.Register(&prefix.TempRolesCommand{})
	registry.Register(&prefix.HelpCommand{Registry: registry})

	appRegistry := commands.NewApplicationRegistry()
	appRegistry.Register(&application.ProfileContextCommand{})
	appRegistry.Register(&application.StaffCommand{})
	appRegistry.Register(&application.BadgeCommand{})
	appRegistry.Register(&application.ReportContextCommand{})
	appRegistry.Register(&application.PanelCommand{})

	db, err := database.Connect(cfg.MongoURI, cfg.DBName)
	if err != nil {
		logger.Error("Failed to connect to database: %v", err)
		return nil
	}

	stateManager := state.NewManager(db)

	return &Bot{
		Session:     dg,
		Config:      cfg,
		Registry:    registry,
		AppRegistry: appRegistry,
		DB:          db,
		State:       stateManager,
	}
}

func (b *Bot) Start() error {
	handler := events.NewEventHandler(b.Registry, b.AppRegistry, b.DB, b.State, b.Config)

	b.Session.AddHandler(handler.OnReady)
	b.Session.AddHandler(handler.OnMessageCreate)
	b.Session.AddHandler(handler.OnInteractionCreate)
	b.Session.AddHandler(handler.OnGuildMemberUpdate)
	b.Session.AddHandler(handler.OnRawEvent)

	handler.StartWorkers(b.Session)

	err := b.Session.Open()
	if err != nil {
		return err
	}

	b.Session.UpdateGameStatus(0, b.Config.Status)

	return nil
}

func (b *Bot) Stop() {
	b.Session.Close()
	if b.DB != nil {
		b.DB.Disconnect()
	}
}
