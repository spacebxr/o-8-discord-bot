package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
)

type Bot struct {
	Session         *discordgo.Session
	DB              *db.Database
	GuildID         string
	RoleHighCommand string
	RoleDevTeam     string
}

func NewBot(token string, database *db.Database, guildID, roleHighCommand, roleDevTeam string) (*Bot, error) {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		Session:         sess,
		DB:              database,
		GuildID:         guildID,
		RoleHighCommand: roleHighCommand,
		RoleDevTeam:     roleDevTeam,
	}

	b.Session.AddHandler(b.ReadyHandler)
	b.Session.AddHandler(b.InteractionCreateHandler)
	b.Session.AddHandler(b.MessageCreateHandler)

	return b, nil
}

func (b *Bot) Start() error {
	err := b.Session.Open()
	if err != nil {
		return err
	}

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "infractioncreate",
			Description: "File an infraction for a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to file an infraction against",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "severity",
					Description: "Severity of the infraction",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for the infraction",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "what",
					Description: "What punishment",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "till_when",
					Description: "Till when they are having the punishment",
					Required:    true,
				},
			},
		},
		{
			Name:        "loarequest",
			Description: "Create a request for leave (LOA)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "from_when",
					Description: "From when (d for day, h for hours, m for minutes)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "till_when",
					Description: "Till when (d for day, h for hours, m for minutes)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for the LOA",
					Required:    true,
				},
			},
		},
		{
			Name:        "roarequest",
			Description: "Create a request for reduced activity (ROA)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "from_when",
					Description: "From when (d for day, h for hours, m for minutes)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "till_when",
					Description: "Till when (d for day, h for hours, m for minutes)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for the ROA",
					Required:    true,
				},
			},
		},
		{
			Name:        "stopwatch",
			Description: "Manage your activity stopwatch",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start the stopwatch",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "stop",
					Description: "Stop the stopwatch",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "status",
					Description: "Check current stopwatch time",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "reset",
					Description: "Reset the stopwatch time",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	_, err = b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, b.GuildID, commands)
	if err != nil {
		log.Printf("Cannot overwrite commands: %v", err)
	}

	return nil
}

func (b *Bot) Stop() {
	b.Session.Close()
}
