package discord

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var deploymentStartTimes = map[string]time.Time{}

func (b *Bot) ReadyHandler(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println("Bot is up!")
}

func (b *Bot) hasAccess(member *discordgo.Member, allowedRoles []string) bool {
	for _, role := range member.Roles {
		for _, devRole := range b.RoleDevTeam {
			if role == devRole {
				return true
			}
		}
		for _, allowed := range allowedRoles {
			if role == allowed {
				return true
			}
		}
	}
	return false
}

func (b *Bot) sendEmbedResponse(s *discordgo.Session, i *discordgo.Interaction, title, description string, color int) {
	s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       color,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
		},
	})
}

func (b *Bot) sendEmbedEphemeral(s *discordgo.Session, i *discordgo.Interaction, title, description string, color int) {
	s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       title,
					Description: description,
					Color:       color,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
		},
	})
}

func (b *Bot) InteractionCreateHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "infractioncreate":
			b.handleInfractionCreateSlash(s, i)
		case "loarequest":
			b.handleLoaCreateSlash(s, i)
		case "roarequest":
			b.handleRoaRequestSlash(s, i)
		case "stopwatch":
			b.handleStopwatchSlash(s, i)
		case "requestcn":
			b.handleRequestCNSlash(s, i)
		case "afk":
			b.handleAFKSlash(s, i)
		case "announce":
			b.handleAnnounceSlash(s, i)
		}
	case discordgo.InteractionMessageComponent:
		cid := i.MessageComponentData().CustomID
		switch cid {
		case "loa_accept", "loa_reject":
			b.handleLoaComponent(s, i)
		case "roa_accept", "roa_reject":
			b.handleRoaComponent(s, i)
		}
		if strings.HasPrefix(cid, "cn_accept_") || strings.HasPrefix(cid, "cn_reject_") {
			b.handleCNComponent(s, i)
		}
		if strings.HasPrefix(cid, "deploy_start_") {
			b.handleDeployStart(s, i)
		}
		if strings.HasPrefix(cid, "deploy_ongoing_") {
			b.handleDeployOngoing(s, i)
		}
		if strings.HasPrefix(cid, "deploy_end_") {
			b.handleDeployEnd(s, i)
		}
	}
}

func (b *Bot) MessageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if author was AFK
	_, _, err := b.DB.GetAFK(context.Background(), m.Author.ID)
	if err == nil {
		b.DB.RemoveAFK(context.Background(), m.Author.ID)
		name := m.Author.GlobalName
		if name == "" {
			name = m.Author.Username
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Welcome back **%s**, your AFK status has been removed.", name))
	}

	// Check mentions
	for _, user := range m.Mentions {
		if user.ID == m.Author.ID {
			continue
		}
		if reason, since, err := b.DB.GetAFK(context.Background(), user.ID); err == nil {
			durationStr := "<t:" + fmt.Sprint(since.Unix()) + ":R>"
			name := user.GlobalName
			if name == "" {
				name = user.Username
			}
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**%s** is currently AFK: **%s** - %s", name, reason, durationStr))
		}
	}

	if strings.HasPrefix(m.Content, "!") {
		args := strings.Fields(m.Content[1:])
		if len(args) == 0 {
			return
		}

		if args[0] == "infractioncreate" {
			s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
				Title:       "Action Required ❗",
				Description: "Please use the `/infractioncreate` slash command.",
				Color:       0xf23f43,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: "https://i.ibb.co/67ZpGxTj/image.png",
				},
			})
		}
	}
}

func (b *Bot) handleInfractionCreateSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied 🚫", "You do not have the required role to file an infraction.", 0xf23f43)
		return
	}

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
		b.sendEmbedResponse(s, i.Interaction, "Error ❌", "Failed to insert infraction into the database.", 0xf23f43)
		return
	}

	count, err := b.DB.CountInfractions(context.Background(), userID)
	if err != nil {
		b.sendEmbedResponse(s, i.Interaction, "Warning ⚠️", "Infraction filed but failed to retrieve total count.", 0xfaa61a)
		return
	}

	responseMsg := fmt.Sprintf("Infraction recorded for <@%s>.\n\n**Severity:** %d\n**Reason:** %s\n**Punishment:** %s\n**Total Infractions:** %d", userID, severity, reason, whatPunishment, count)
	b.sendEmbedResponse(s, i.Interaction, "Infraction Filed ✅", responseMsg, 0x23a559)
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

	content := fmt.Sprintf("<@%s> is requesting a Leave of Absence.\n\n**From:** %s\n**To:** %s\n**Reason:** %s", i.Member.User.ID, fromWhen, tillWhen, reason)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "LOA Request 📝",
					Description: content,
					Color:       0x5865F2,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
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
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied 🚫", "You do not have the required high command role to action this LOA.", 0xf23f43)
		return
	}

	action := "Accepted"
	color := 0x23a559
	if i.MessageComponentData().CustomID == "loa_reject" {
		action = "Rejected"
		color = 0xf23f43
	}

	embed := i.Message.Embeds[0]
	embed.Title = "LOA Request - " + action
	embed.Color = color
	embed.Description += fmt.Sprintf("\n\n**Status:** %s by <@%s>", action, i.Member.User.ID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (b *Bot) handleRoaRequestSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	content := fmt.Sprintf("<@%s> is requesting Reduced on Activity.\n\n**From:** %s\n**To:** %s\n**Reason:** %s", i.Member.User.ID, fromWhen, tillWhen, reason)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "ROA Request 📝",
					Description: content,
					Color:       0xEBb026,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
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
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied 🚫", "You do not have the required high command role to action this ROA.", 0xf23f43)
		return
	}

	action := "Accepted"
	color := 0x23a559
	if i.MessageComponentData().CustomID == "roa_reject" {
		action = "Rejected"
		color = 0xf23f43
	}

	embed := i.Message.Embeds[0]
	embed.Title = "ROA Request - " + action
	embed.Color = color
	embed.Description += fmt.Sprintf("\n\n**Status:** %s by <@%s>", action, i.Member.User.ID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (b *Bot) handleStopwatchSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	subcommand := options[0].Name

	ctx := context.Background()
	userID := i.Member.User.ID
	var title, content string
	color := 0x5865F2

	switch subcommand {
	case "start":
		title = "Stopwatch Started ⏱️"
		err := b.DB.StartStopwatch(ctx, userID)
		if err != nil {
			content = "Failed to start stopwatch. It might already be running."
			color = 0xf23f43
		} else {
			content = "Your activity stopwatch is now running."
			color = 0x23a559
		}
	case "stop":
		title = "Stopwatch Stopped ⏹️"
		total, err := b.DB.StopStopwatch(ctx, userID)
		if err != nil {
			content = "Failed to stop stopwatch. It might not be running."
			color = 0xf23f43
		} else {
			content = fmt.Sprintf("Stopwatch stopped!\n**Session/Total Time:** %s", formatDuration(total))
			color = 0x23a559
		}
	case "status":
		title = "Stopwatch Status 📊"
		startTime, total, err := b.DB.GetStopwatch(ctx, userID)
		if err != nil {
			content = "No recorded stopwatch data found."
			color = 0xf23f43
		} else {
			current := total
			status := "Stopped"
			if startTime != nil {
				current += int64(time.Since(*startTime).Seconds())
				status = "Running"
			}
			content = fmt.Sprintf("Current Status: **%s**\nAccumulated Time: **%s**", status, formatDuration(current))
		}
	case "reset":
		title = "Stopwatch Reset ♻️"
		err := b.DB.ResetStopwatch(ctx, userID)
		if err != nil {
			content = "Failed to reset stopwatch data."
			color = 0xf23f43
		} else {
			content = "Your activity stopwatch has been reset to zero."
			color = 0x23a559
		}
	}

	b.sendEmbedResponse(s, i.Interaction, title, content, color)
}

func formatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (b *Bot) handleRequestCNSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var robloxUsername string
	var codename string

	for _, opt := range options {
		switch opt.Name {
		case "roblox_username":
			robloxUsername = opt.StringValue()
		case "codename":
			codename = opt.StringValue()
		}
	}

	taken, err := b.DB.IsCodenameTaken(context.Background(), codename)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Database Error ❌", "An error occurred while checking if the codename was taken.", 0xf23f43)
		return
	}
	if taken {
		b.sendEmbedEphemeral(s, i.Interaction, "Codename Unavailable ⚠️", fmt.Sprintf("The codename **%s** is already taken by an approved user. Please request a different codename.", codename), 0xf23f43)
		return
	}

	reqID, err := b.DB.InsertCodenameRequest(context.Background(), i.Member.User.ID, robloxUsername, codename)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Database Error ❌", "Failed to save your codename request.", 0xf23f43)
		return
	}

	content := fmt.Sprintf("<@%s> is requesting a new Codename.\n\n**Roblox Username:** %s\n**Desired Codename:** %s", i.Member.User.ID, robloxUsername, codename)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Codename Request 🏷️",
					Description: content,
					Color:       0x5865F2,
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Approve",
							Style:    discordgo.SuccessButton,
							CustomID: "cn_accept_" + reqID,
						},
						discordgo.Button{
							Label:    "Deny",
							Style:    discordgo.DangerButton,
							CustomID: "cn_reject_" + reqID,
						},
					},
				},
			},
		},
	})
}

func (b *Bot) handleCNComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied 🚫", "You do not have the required high command role to action this Codename request.", 0xf23f43)
		return
	}

	customID := i.MessageComponentData().CustomID
	var action, status string
	var color int

	var reqID string
	if strings.HasPrefix(customID, "cn_accept_") {
		action = "Approved"
		status = "approved"
		color = 0x23a559
		reqID = strings.TrimPrefix(customID, "cn_accept_")
	} else if strings.HasPrefix(customID, "cn_reject_") {
		action = "Denied"
		status = "denied"
		color = 0xf23f43
		reqID = strings.TrimPrefix(customID, "cn_reject_")
	} else {
		return
	}

	err := b.DB.UpdateCodenameStatus(context.Background(), reqID, status)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Database Error ❌", "Failed to update codename status.", 0xf23f43)
		return
	}

	embed := i.Message.Embeds[0]
	embed.Title = "Codename Request - " + action
	embed.Color = color
	embed.Description += fmt.Sprintf("\n\n**Status:** %s by <@%s>", action, i.Member.User.ID)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (b *Bot) handleAFKSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	reason := "AFK"

	if len(options) > 0 {
		reason = options[0].StringValue()
	}

	var userID string
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	if userID == "" {
		b.sendEmbedEphemeral(s, i.Interaction, "Error ❌", "User identification failed.", 0xf23f43)
		return
	}

	err := b.DB.SetAFK(context.Background(), userID, reason)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Error ❌", "Failed to set AFK status.", 0xf23f43)
		return
	}

	b.sendEmbedResponse(s, i.Interaction, "AFK Status 💤", fmt.Sprintf("You are now AFK: **%s**", reason), 0x23a559)
}

func (b *Bot) handleAnnounceSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied", "You do not have the required role to make announcements.", 0xf23f43)
		return
	}

	sub := i.ApplicationCommandData().Options[0]
	switch sub.Name {
	case "message":
		b.handleAnnounceMessage(s, i, sub)
	case "deployment":
		b.handleAnnounceDeployment(s, i, sub)
	}
}

func (b *Bot) handleAnnounceMessage(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	var text string
	for _, opt := range sub.Options {
		if opt.Name == "text" {
			text = opt.StringValue()
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Announcement",
					Description: text,
					Color:       0x5865F2,
					Footer: &discordgo.MessageEmbedFooter{
						Text: "Announced by " + i.Member.User.Username,
					},
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
		},
	})
}

func (b *Bot) handleAnnounceDeployment(s *discordgo.Session, i *discordgo.InteractionCreate, sub *discordgo.ApplicationCommandInteractionDataOption) {
	var message, participants, location string
	var hostUser, cohostUser *discordgo.User

	for _, opt := range sub.Options {
		switch opt.Name {
		case "message":
			message = opt.StringValue()
		case "host":
			hostUser = opt.UserValue(s)
		case "cohost":
			cohostUser = opt.UserValue(s)
		case "participants":
			participants = opt.StringValue()
		case "location":
			location = opt.StringValue()
		}
	}

	hostMention := "Unknown"
	if hostUser != nil {
		hostMention = "<@" + hostUser.ID + ">"
	}
	cohostMention := "Unknown"
	if cohostUser != nil {
		cohostMention = "<@" + cohostUser.ID + ">"
	}

	description := fmt.Sprintf(
		"%s\n\n**Host:** %s\n**Co-Host:** %s\n**Participants:** %s\n**Location:** %s",
		message, hostMention, cohostMention, participants, location,
	)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Deployment Scheduled",
					Description: description,
					Color:       0xFAA61A,
					Timestamp:   time.Now().UTC().Format(time.RFC3339),
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://i.ibb.co/67ZpGxTj/image.png",
					},
				},
			},
		},
	})
	if err != nil {
		return
	}

	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		return
	}

	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: msg.ChannelID,
		ID:      msg.ID,
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Deployment Scheduled",
				Description: description,
				Color:       0xFAA61A,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: "https://i.ibb.co/67ZpGxTj/image.png",
				},
			},
		},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Start Deployment",
						Style:    discordgo.SuccessButton,
						CustomID: "deploy_start_" + msg.ID,
					},
				},
			},
		},
	})
}

func (b *Bot) handleDeployStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied", "You do not have permission to manage deployments.", 0xf23f43)
		return
	}

	msgID := strings.TrimPrefix(i.MessageComponentData().CustomID, "deploy_start_")
	startTime := time.Now().UTC()
	deploymentStartTimes[msgID] = startTime

	embed := i.Message.Embeds[0]
	embed.Title = "Deployment Started"
	embed.Color = 0x23a559
	embed.Description += fmt.Sprintf("\n\n**Started:** <t:%d:F>", startTime.Unix())

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Status: Ongoing",
							Style:    discordgo.PrimaryButton,
							CustomID: "deploy_ongoing_" + msgID,
						},
					},
				},
			},
		},
	})
}

func (b *Bot) handleDeployOngoing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied", "You do not have permission to manage deployments.", 0xf23f43)
		return
	}

	msgID := strings.TrimPrefix(i.MessageComponentData().CustomID, "deploy_ongoing_")

	embed := i.Message.Embeds[0]
	embed.Title = "Deployment Ongoing"
	embed.Color = 0x5865F2

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "End Deployment",
							Style:    discordgo.DangerButton,
							CustomID: "deploy_end_" + msgID,
						},
					},
				},
			},
		},
	})
}

func (b *Bot) handleDeployEnd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.hasAccess(i.Member, b.RoleHighCommand) {
		b.sendEmbedEphemeral(s, i.Interaction, "Access Denied", "You do not have permission to manage deployments.", 0xf23f43)
		return
	}

	msgID := strings.TrimPrefix(i.MessageComponentData().CustomID, "deploy_end_")
	endTime := time.Now().UTC()

	var durationStr string
	if startTime, ok := deploymentStartTimes[msgID]; ok {
		duration := endTime.Sub(startTime)
		h := int(duration.Hours())
		m := int(duration.Minutes()) % 60
		sec := int(duration.Seconds()) % 60
		durationStr = fmt.Sprintf("%02dh %02dm %02ds", h, m, sec)
		delete(deploymentStartTimes, msgID)
	} else {
		durationStr = "Unknown"
	}

	embed := i.Message.Embeds[0]
	embed.Title = "Deployment Ended"
	embed.Color = 0xf23f43

	var startDisplay string
	for _, field := range embed.Fields {
		if field.Name == "Started" {
			startDisplay = field.Value
		}
	}

	if startDisplay == "" {
		if startTime, ok := deploymentStartTimes[msgID]; ok {
			startDisplay = fmt.Sprintf("<t:%d:F>", startTime.Unix())
		}
	}

	embed.Fields = append(embed.Fields,
		&discordgo.MessageEmbedField{
			Name:   "End Time",
			Value:  fmt.Sprintf("<t:%d:F>", endTime.Unix()),
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name:   "Duration",
			Value:  durationStr,
			Inline: true,
		},
	)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}
