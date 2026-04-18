package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Bot is up!")
}

func (b *Bot) InteractionCreateHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "infractioncreate":
			b.handleInfractionCreateSlash(s, i)
		case "loacreate":
			b.handleLoaCreateSlash(s, i)
		case "roacreate":
			b.handleRoaCreateSlash(s, i)
		case "stopwatch":
			b.handleStopwatchSlash(s, i)
		}
	case discordgo.InteractionMessageComponent:
		switch i.MessageComponentData().CustomID {
		case "loa_accept", "loa_reject":
			b.handleLoaComponent(s, i)
		case "roa_accept", "roa_reject":
			b.handleRoaComponent(s, i)
		}
	}
}

func (b *Bot) MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		args := strings.Fields(m.Content[1:])
		if len(args) == 0 {
			return
		}

		if args[0] == "infractioncreate" {
			s.ChannelMessageSend(m.ChannelID, "Please use the /infractioncreate slash command.")
		}
	}
}

func (b *Bot) handleInfractionCreateSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User
	var severity int64
	var reason string
	var whatPunishment string
	var tillWhen string

	for _, opt := range options {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "severity":
			severity = opt.IntValue()
		case "reason":
			reason = opt.StringValue()
		case "what":
			whatPunishment = opt.StringValue()
		case "till_when":
			tillWhen = opt.StringValue()
		}
	}

	modID := i.Member.User.ID
	userID := targetUser.ID

	err := b.DB.InsertInfraction(context.Background(), userID, modID, int(severity), reason, whatPunishment, tillWhen)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to insert infraction.",
			},
		})
		return
	}

	count, err := b.DB.CountInfractions(context.Background(), userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Infraction filed but failed to count.",
			},
		})
		return
	}

	responseMsg := fmt.Sprintf("Infraction filed for <@%s>. Total infractions: %d", userID, count)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseMsg,
		},
	})
}

func (b *Bot) handleLoaCreateSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var fromWhen string
	var tillWhen string
	var reason string

	for _, opt := range options {
		switch opt.Name {
		case "from_when":
			fromWhen = opt.StringValue()
		case "till_when":
			tillWhen = opt.StringValue()
		case "reason":
			reason = opt.StringValue()
		}
	}

	content := fmt.Sprintf("<@%s> requested a Leave of Absence from **%s** till **%s**\n**Reason:** %s", i.Member.User.ID, fromWhen, tillWhen, reason)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Accept",
							Style:    discordgo.SuccessButton,
							CustomID: "loa_accept",
						},
						discordgo.Button{
							Label:    "Reject",
							Style:    discordgo.DangerButton,
							CustomID: "loa_reject",
						},
					},
				},
			},
		},
	})
}

func (b *Bot) handleLoaComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	hasPerm := false
	for _, role := range i.Member.Roles {
		if role == b.RoleHighCommand {
			hasPerm = true
			break
		}
	}

	if !hasPerm {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You do not have the required high command role to action this LOA.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	action := "Accepted"
	if i.MessageComponentData().CustomID == "loa_reject" {
		action = "Rejected"
	}

	oldContent := i.Message.Content
	newContent := fmt.Sprintf("%s\n\n**Status:** %s by <@%s>", oldContent, action, i.Member.User.ID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    newContent,
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (b *Bot) handleRoaCreateSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var fromWhen string
	var tillWhen string
	var reason string

	for _, opt := range options {
		switch opt.Name {
		case "from_when":
			fromWhen = opt.StringValue()
		case "till_when":
			tillWhen = opt.StringValue()
		case "reason":
			reason = opt.StringValue()
		}
	}

	content := fmt.Sprintf("<@%s> requested a Reduced on Activity from **%s** till **%s**\n**Reason:** %s", i.Member.User.ID, fromWhen, tillWhen, reason)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Accept",
							Style:    discordgo.SuccessButton,
							CustomID: "roa_accept",
						},
						discordgo.Button{
							Label:    "Reject",
							Style:    discordgo.DangerButton,
							CustomID: "roa_reject",
						},
					},
				},
			},
		},
	})
}

func (b *Bot) handleRoaComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	hasPerm := false
	for _, role := range i.Member.Roles {
		if role == b.RoleHighCommand {
			hasPerm = true
			break
		}
	}

	if !hasPerm {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You do not have the required high command role to action this ROA.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	action := "Accepted"
	if i.MessageComponentData().CustomID == "roa_reject" {
		action = "Rejected"
	}

	oldContent := i.Message.Content
	newContent := fmt.Sprintf("%s\n\n**Status:** %s by <@%s>", oldContent, action, i.Member.User.ID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    newContent,
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (b *Bot) handleStopwatchSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	subcommand := options[0].Name

	ctx := context.Background()
	userID := i.Member.User.ID
	var content string

	switch subcommand {
	case "start":
		err := b.DB.StartStopwatch(ctx, userID)
		if err != nil {
			content = "Failed to start stopwatch or it is already running."
		} else {
			content = "Stopwatch started!"
		}
	case "stop":
		total, err := b.DB.StopStopwatch(ctx, userID)
		if err != nil {
			content = "Failed to stop stopwatch or it is not running."
		} else {
			content = fmt.Sprintf("Stopwatch stopped! Total time: %s", formatDuration(total))
		}
	case "status":
		startTime, total, err := b.DB.GetStopwatch(ctx, userID)
		if err != nil {
			content = "No recorded stopwatch data found."
		} else {
			current := total
			status := "Stopped"
			if startTime != nil {
				current += int64(time.Since(*startTime).Seconds())
				status = "Running"
			}
			content = fmt.Sprintf("Status: **%s**\nTotal Time: **%s**", status, formatDuration(current))
		}
	case "reset":
		err := b.DB.ResetStopwatch(ctx, userID)
		if err != nil {
			content = "Failed to reset stopwatch."
		} else {
			content = "Stopwatch reset!"
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func formatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
