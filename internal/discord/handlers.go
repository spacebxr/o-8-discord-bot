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
	if i.Type == discordgo.InteractionApplicationCommand {
		if i.ApplicationCommandData().Name == "file-incident" {
			b.handleFileIncidentSlash(s, i)
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

		if args[0] == "file-incident" {
			s.ChannelMessageSend(m.ChannelID, "Please use the /file-incident slash command.")
		}
	}
}

func (b *Bot) handleFileIncidentSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	var targetUser *discordgo.User
	var severity int64
	var reason string

	for _, opt := range options {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "severity":
			severity = opt.IntValue()
		case "reason":
			reason = opt.StringValue()
		}
	}

	modID := i.Member.User.ID
	userID := targetUser.ID

	err := b.DB.InsertInfraction(context.Background(), userID, modID, int(severity), reason)
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
