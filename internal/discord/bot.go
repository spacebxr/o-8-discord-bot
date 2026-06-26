package discord

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
	"github.com/spacebxr/o-8-discord-bot/internal/storage"
)

type Bot struct {
	Session         *discordgo.Session
	DB              *db.Database
	GuildID         string
	RoleHighCommand []string
	RoleDevTeam     []string
	Storage        *storage.Client
	VoiceManager    *VoiceManager
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

func NewBot(token string, database *db.Database, storageClient *storage.Client, guildID, roleHighCommand, roleDevTeam string) (*Bot, error) {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	sess.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsGuildVoiceStates

	b := &Bot{
		Session:         sess,
		DB:              database,
		GuildID:         guildID,
		RoleHighCommand: splitRoles(roleHighCommand),
		RoleDevTeam:     splitRoles(roleDevTeam),
		Storage:        storageClient,
		VoiceManager:    NewVoiceManager(),
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
			Name:        "syncroles",
			Description: "Sync all member roles and metadata to the database",
		},
		{
			Name:        "loarequest",
			Description: "Create a request for leave (LOA)",
			Options: []*discordgo.ApplicationCommandOption{
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
			Name:        "changecn",
			Description: "Change a user's codename (CN)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "The user whose codename to change",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "codename",
					Description: "The new codename",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "roblox_username",
					Description: "The user's Roblox username",
					Required:    false,
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
			Name:        "vc",
			Description: "Voice channel controls",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "panel",
					Description: "Show the voice control panel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "join",
					Description: "Join your voice channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "leave",
					Description: "Leave the voice channel",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		{
			Name:        "schedule",
			Description: "Schedule messages for later delivery",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "create",
					Description: "Create a scheduled message",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionChannel,
							Name:        "channel",
							Description: "The channel to send the message in",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "content",
							Description: "The message content",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "delay",
							Description: "When to send (e.g. 30m, 2h, 1d)",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "repeat",
							Description: "How often to repeat",
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "Minutes", Value: "minutes"},
								{Name: "Hours", Value: "hours"},
								{Name: "Days", Value: "days"},
								{Name: "Weeks", Value: "weeks"},
								{Name: "Months", Value: "months"},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "repeat_interval",
							Description: "Repeat interval (default: 1)",
							Required:    false,
						},
					},
				},
				{
					Name:        "list",
					Description: "List upcoming scheduled messages",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
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
	go b.startPersonnelSync()
	go b.startScheduledMessageChecker()

	return nil
}

func (b *Bot) Stop() {
	b.Session.Close()
}
