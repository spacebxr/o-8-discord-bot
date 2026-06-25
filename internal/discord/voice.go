package discord

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/hraban/opus.v2"
)

type PlaybackState int

const (
	PlaybackStopped PlaybackState = iota
	PlaybackPlaying
	PlaybackPaused
)

type VoiceSession struct {
	VoiceConnection *discordgo.VoiceConnection
	GuildID         string
	ChannelID       string
	RecordingName   string
	PlaybackState   PlaybackState
	stopChan        chan struct{}
	mu              sync.Mutex
}

type VoiceManager struct {
	mu       sync.Mutex
	sessions map[string]*VoiceSession // guildID -> session
}

func NewVoiceManager() *VoiceManager {
	return &VoiceManager{
		sessions: make(map[string]*VoiceSession),
	}
}

func (vm *VoiceManager) GetSession(guildID string) *VoiceSession {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	return vm.sessions[guildID]
}

func (vm *VoiceManager) StopSession(guildID string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	if s, ok := vm.sessions[guildID]; ok {
		close(s.stopChan)
		if s.VoiceConnection != nil {
			s.VoiceConnection.Disconnect()
		}
		delete(vm.sessions, guildID)
	}
}

func (vm *VoiceManager) SetSession(guildID string, session *VoiceSession) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.sessions[guildID] = session
}

func playAudioFile(vc *discordgo.VoiceConnection, fileURL string, stopChan chan struct{}) error {
	vc.Speaking(true)
	defer vc.Speaking(false)

	// For R2 URLs or direct URLs, download to a temp file first
	var inputPath string
	if fileURL != "" {
		// Check if it's a local file or needs downloading
		if _, err := os.Stat(fileURL); err == nil {
			inputPath = fileURL
		} else {
			// Download to temp file
			tmpFile, err := os.CreateTemp("", "audio-*.mp3")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			// Use curl or wget to download (we have wget in alpine)
			cmd := exec.Command("wget", "-q", "-O", tmpFile.Name(), fileURL)
			if err := cmd.Run(); err != nil {
				// Try curl
				cmd = exec.Command("curl", "-s", "-o", tmpFile.Name(), fileURL)
				if err := cmd.Run(); err != nil {
					return fmt.Errorf("failed to download audio file: %w", err)
				}
			}
			inputPath = tmpFile.Name()
		}
	} else {
		return fmt.Errorf("no file URL provided")
	}

	// Spawn FFmpeg: decode to PCM s16le 48000Hz mono
	ffmpegCmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-f", "s16le",
		"-ac", "1",
		"-ar", "48000",
		"pipe:1",
	)

	stdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create ffmpeg pipe: %w", err)
	}
	stderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create ffmpeg stderr pipe: %w", err)
	}

	if err := ffmpegCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	defer func() {
		ffmpegCmd.Process.Kill()
		ffmpegCmd.Wait()
	}()

	// Drain stderr to avoid blocking
	go io.Copy(io.Discard, stderr)

	// Create Opus encoder: 48000Hz, mono, voice optimized
	enc, err := opus.NewEncoder(48000, 1, opus.AppVoIP)
	if err != nil {
		return fmt.Errorf("failed to create opus encoder: %w", err)
	}

	// Read PCM and encode to Opus in 20ms frames (960 samples @ 48kHz)
	const frameSize = 960      // samples per frame (20ms)
	const pcmBufSize = frameSize * 2 // 2 bytes per sample (s16le)

	pcmBuf := make([]byte, pcmBufSize)
	opusBuf := make([]byte, 1500) // max opus frame size

	reader := bufio.NewReaderSize(stdout, pcmBufSize*64)

	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		_, err := io.ReadFull(reader, pcmBuf)
		if err != nil {
			break
		}

		// Convert s16le bytes to []int16
		pcm := make([]int16, frameSize)
		for i := range pcm {
			pcm[i] = int16(binary.LittleEndian.Uint16(pcmBuf[i*2:]))
		}

		// Encode to Opus
		n, err := enc.Encode(pcm, opusBuf)
		if err != nil {
			continue
		}

		select {
		case vc.OpusSend <- opusBuf[:n]:
		case <-stopChan:
			return nil
		case <-time.After(time.Second):
			return fmt.Errorf("timed out sending opus frame")
		}
	}

	return nil
}

func (b *Bot) joinVoiceChannel(s *discordgo.Session, guildID, channelID string) (*discordgo.VoiceConnection, error) {
	// Leave any existing connection in this guild
	b.VoiceManager.StopSession(guildID)

	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel: %w", err)
	}

	// Wait for connection to be ready
	vc.Ready = true
	return vc, nil
}

func (b *Bot) getVoiceChannelID(s *discordgo.Session, guildID, userID string) string {
	g, err := s.State.Guild(guildID)
	if err != nil {
		return ""
	}
	for _, vs := range g.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID
		}
	}
	return ""
}

func (b *Bot) buildVoicePanel(guildID string) (*discordgo.MessageEmbed, discordgo.MessageComponent, discordgo.MessageComponent) {
	ctx := context.Background()
	recordings, err := b.DB.GetAudioRecordings(ctx)
	if err != nil {
		recordings = nil
	}

	session := b.VoiceManager.GetSession(guildID)

	embed := &discordgo.MessageEmbed{
		Title:       "Voice Control Panel",
		Description: "Select a recording from the dropdown below to play it in your voice channel.",
		Color:       0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.ibb.co/67ZpGxTj/image.png",
		},
	}

	if session != nil && session.PlaybackState == PlaybackPlaying {
		embed.Title = "Now Playing"
		embed.Description = fmt.Sprintf("**%s** is currently playing.", session.RecordingName)
	}

	var selectMenu discordgo.MessageComponent
	if len(recordings) > 0 {
		var options []discordgo.SelectMenuOption
		for _, r := range recordings {
			options = append(options, discordgo.SelectMenuOption{
				Label: r.Name,
				Value: r.ID,
				Description: fmt.Sprintf("%.0f seconds", r.DurationSeconds),
			})
		}
		selectMenu = discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					MenuType:    discordgo.StringSelectMenu,
					CustomID:    "vc_select",
					Placeholder: "Choose a recording to play...",
					Options:     options,
				},
			},
		}
	} else {
		embed.Description = "No recordings available. Upload recordings from the web dashboard first."
	}

	stopButton := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Stop",
				Style:    discordgo.DangerButton,
				CustomID: "vc_stop_stop",
				Disabled: session == nil || session.PlaybackState != PlaybackPlaying,
			},
			discordgo.Button{
				Label:    "Refresh",
				Style:    discordgo.SecondaryButton,
				CustomID: "vc_refresh",
			},
		},
	}

	return embed, selectMenu, stopButton
}

func (b *Bot) handleVoiceSlash(s *discordgo.Session, i *discordgo.InteractionCreate) {
	sub := i.ApplicationCommandData().Options[0]
	switch sub.Name {
	case "panel":
		b.handleVoicePanel(s, i)
	case "join":
		b.handleVoiceJoin(s, i)
	case "leave":
		b.handleVoiceLeave(s, i)
	}
}

func (b *Bot) handleVoicePanel(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed, selectMenu, stopButton := b.buildVoicePanel(i.GuildID)
	components := []discordgo.MessageComponent{}
	if selectMenu != nil {
		components = append(components, selectMenu)
	}
	components = append(components, stopButton)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func (b *Bot) handleVoiceJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := b.getVoiceChannelID(s, i.GuildID, i.Member.User.ID)
	if channelID == "" {
		b.sendEmbedEphemeral(s, i.Interaction, "Not in a Voice Channel", "You need to be in a voice channel for me to join.", 0xf23f43)
		return
	}

	vc, err := b.joinVoiceChannel(s, i.GuildID, channelID)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Error", "Failed to join voice channel: "+err.Error(), 0xf23f43)
		return
	}

	b.VoiceManager.SetSession(i.GuildID, &VoiceSession{
		VoiceConnection: vc,
		GuildID:         i.GuildID,
		ChannelID:       channelID,
		PlaybackState:   PlaybackStopped,
		stopChan:        make(chan struct{}),
	})

	b.sendEmbedEphemeral(s, i.Interaction, "Joined Voice Channel", "I've joined your voice channel. Use `/vc panel` to play recordings.", 0x23a559)
}

func (b *Bot) handleVoiceLeave(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.VoiceManager.StopSession(i.GuildID)
	b.sendEmbedEphemeral(s, i.Interaction, "Left Voice Channel", "I've left the voice channel.", 0xf23f43)
}

func (b *Bot) handleVoiceSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	recordingID := i.MessageComponentData().Values[0]

	recording, err := b.DB.GetAudioRecordingByID(context.Background(), recordingID)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Error", "Recording not found.", 0xf23f43)
		return
	}

	channelID := b.getVoiceChannelID(s, i.GuildID, i.Member.User.ID)
	if channelID == "" {
		b.sendEmbedEphemeral(s, i.Interaction, "Not in a Voice Channel", "You need to be in a voice channel first.", 0xf23f43)
		return
	}

	b.VoiceManager.StopSession(i.GuildID)

	vc, err := b.joinVoiceChannel(s, i.GuildID, channelID)
	if err != nil {
		b.sendEmbedEphemeral(s, i.Interaction, "Error", "Failed to join voice channel.", 0xf23f43)
		return
	}

	stopChan := make(chan struct{})
	session := &VoiceSession{
		VoiceConnection: vc,
		GuildID:         i.GuildID,
		ChannelID:       channelID,
		RecordingName:   recording.Name,
		PlaybackState:   PlaybackPlaying,
		stopChan:        stopChan,
	}
	b.VoiceManager.SetSession(i.GuildID, session)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	go func() {
		fileURL := recording.FileURL
		if b.Storage != nil {
			if u, err := b.Storage.GetURL(context.Background(), recording.FileURL); err == nil {
				fileURL = u
			}
		}
		playErr := playAudioFile(vc, fileURL, stopChan)
		session.mu.Lock()
		session.PlaybackState = PlaybackStopped
		session.mu.Unlock()
		if playErr != nil {
			b.VoiceManager.StopSession(i.GuildID)
		}
	}()

	embed, selectMenuComp, stopButtonComp := b.buildVoicePanel(i.GuildID)
	components := []discordgo.MessageComponent{}
	if selectMenuComp != nil {
		components = append(components, selectMenuComp)
	}
	components = append(components, stopButtonComp)

	s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    i.ChannelID,
		ID:         i.Message.ID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}

func (b *Bot) handleVoicePlayButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.sendEmbedEphemeral(s, i.Interaction, "Use the Dropdown", "Select a recording from the dropdown menu above to play it.", 0xfaa61a)
}

func (b *Bot) handleVoiceStopButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.VoiceManager.StopSession(i.GuildID)

	embed, selectMenuComp, stopButtonComp := b.buildVoicePanel(i.GuildID)
	components := []discordgo.MessageComponent{}
	if selectMenuComp != nil {
		components = append(components, selectMenuComp)
	}
	components = append(components, stopButtonComp)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func (b *Bot) handleVoiceRefreshButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed, selectMenuComp, stopButtonComp := b.buildVoicePanel(i.GuildID)
	components := []discordgo.MessageComponent{}
	if selectMenuComp != nil {
		components = append(components, selectMenuComp)
	}
	components = append(components, stopButtonComp)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}
