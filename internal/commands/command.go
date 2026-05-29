package commands

import (
	"discord-bot/internal/config"
	"discord-bot/internal/database"
	"discord-bot/internal/state"

	"github.com/bwmarrin/discordgo"
)

type Context struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
	Args    []string
	DB      *database.Database
	State   *state.Manager
	Config  *config.Config
}

type Command interface {
	Name() string
	Aliases() []string
	Description() string
	Execute(ctx *Context) error
}

type Registry struct {
	Prefix   string
	Commands map[string]Command
}

func NewRegistry(prefix string) *Registry {
	return &Registry{
		Prefix:   prefix,
		Commands: make(map[string]Command),
	}
}

func (r *Registry) Register(cmd Command) {
	r.Commands[cmd.Name()] = cmd
	for _, alias := range cmd.Aliases() {
		r.Commands[alias] = cmd
	}
}

type ApplicationContext struct {
	Session     *discordgo.Session
	Interaction *discordgo.InteractionCreate
	DB          *database.Database
	State       *state.Manager
	Config      *config.Config
}

type ApplicationCommand interface {
	Definition() *discordgo.ApplicationCommand
	Execute(ctx *ApplicationContext) error
}

type ApplicationRegistry struct {
	Commands map[string]ApplicationCommand
}

func NewApplicationRegistry() *ApplicationRegistry {
	return &ApplicationRegistry{
		Commands: make(map[string]ApplicationCommand),
	}
}

func (r *ApplicationRegistry) Register(cmd ApplicationCommand) {
	r.Commands[cmd.Definition().Name] = cmd
}
