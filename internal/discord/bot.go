package discord

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
)

type Bot struct {
	Session         *discordgo.Session
	DB              *db.Database
	GuildID         string
	RoleHighCommand []string
	RoleDevTeam     []string
}

func splitRoles(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
		RoleHighCommand: splitRoles(roleHighCommand),
		RoleDevTeam:     splitRoles(roleDevTeam),
	}

	b.Session.AddHandler(b.ReadyHandler)
	b.Session.AddHandler(b.InteractionCreateHandler)
	b.Session.AddHandler(b.MessageCreateHandler)
	b.Session.AddHandler(b.MessageReactionAddHandler)
	b.Session.AddHandler(b.MessageReactionRemoveHandler)
	b.Session.AddHandler(b.GuildMemberUpdateHandler)

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
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "punishment",
					Description: "The punishment applied",
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
					Name:        "appeal_due",
					Description: "Due date of appeal",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionAttachment,
					Name:        "image",
					Description: "Optional image attachment for proof",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "add_role",
					Description: "Role to add to the user",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "remove_role",
					Description: "Role to remove from the user",
					Required:    false,
				},
			},
		},
		{
			Name:        "infractionhistory",
			Description: "Search infraction history for a user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user to get infractions for",
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
		{
			Name:        "requestcn",
			Description: "Request a codename (CN)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "roblox_username",
					Description: "Your Roblox username",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "codename",
					Description: "Your desired codename",
					Required:    true,
				},
			},
		},
		{
			Name:        "afk",
			Description: "Set your AFK status",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "Reason for being AFK",
					Required:    false,
				},
			},
		},
		{
			Name:        "announce",
			Description: "Make an announcement",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "message",
					Description: "Send a normal announcement message",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "text",
							Description: "The announcement message",
							Required:    true,
						},
					},
				},
				{
					Name:        "deployment",
					Description: "Schedule a deployment",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "message",
							Description: "Deployment details / message",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "host",
							Description: "Host of the deployment",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "cohost",
							Description: "Co-host of the deployment",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "participants",
							Description: "Mention participants (@user @user ...)",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "location",
							Description: "Location of the deployment",
							Required:    true,
						},
					},
				},
			},
		},
	}

	_, err = b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, b.GuildID, commands)
	if err != nil {
		log.Printf("Cannot overwrite commands: %v", err)
	}

	go b.startExpiryChecker()

	return nil
}

func (b *Bot) Stop() {
	b.Session.Close()
}
