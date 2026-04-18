package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
)

type Bot struct {
	Session    *discordgo.Session
	DB         *db.Database
	GuildID    string
	RoleL4          string
	RoleClassD      string
	RoleHighCommand string
}

func NewBot(token string, database *db.Database, guildID, roleL4, roleClassD, roleHighCommand string) (*Bot, error) {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		Session:         sess,
		DB:              database,
		GuildID:         guildID,
		RoleL4:          roleL4,
		RoleClassD:      roleClassD,
		RoleHighCommand: roleHighCommand,
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
			Name:        "loacreate",
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
			},
		},
	}

	for _, cmd := range commands {
		_, err := b.Session.ApplicationCommandCreate(b.Session.State.User.ID, b.GuildID, cmd)
		if err != nil {
			log.Printf("Cannot create '%v' command: %v", cmd.Name, err)
		}
	}

	return nil
}

func (b *Bot) Stop() {
	b.Session.Close()
}
