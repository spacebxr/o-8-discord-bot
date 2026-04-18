package discord

import (
	"context"
	"fmt"
	"strings"

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
		}
	case discordgo.InteractionMessageComponent:
		switch i.MessageComponentData().CustomID {
		case "loa_accept", "loa_reject":
			b.handleLoaComponent(s, i)
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

	if count >= 3 {
		errAdd := s.GuildMemberRoleAdd(b.GuildID, userID, b.RoleClassD)
		errRem := s.GuildMemberRoleRemove(b.GuildID, userID, b.RoleL4)
		if errAdd != nil || errRem != nil {
			responseMsg += "\nFailed to update roles."
		} else {
			responseMsg += "\nUser downgraded to Class-D."
		}
	}

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

	for _, opt := range options {
		switch opt.Name {
		case "from_when":
			fromWhen = opt.StringValue()
		case "till_when":
			tillWhen = opt.StringValue()
		}
	}

	content := fmt.Sprintf("<@%s> requested a Leave of Absence from **%s** till **%s**", i.Member.User.ID, fromWhen, tillWhen)

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
