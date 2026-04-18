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
	RoleL4     string
	RoleClassD string
}

func NewBot(token string, database *db.Database, guildID, roleL4, roleClassD string) (*Bot, error) {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		Session:    sess,
		DB:         database,
		GuildID:    guildID,
		RoleL4:     roleL4,
		RoleClassD: roleClassD,
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
			Name:        "file-incident",
			Description: "File an incident for a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to file an incident against",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "severity",
					Description: "Severity of the incident",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for the incident",
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
