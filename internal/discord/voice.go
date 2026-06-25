package discord

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
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
			s.VoiceConnection.Disconnect(context.Background())
		}
		delete(vm.sessions, guildID)
	}
}

func (vm *VoiceManager) SetSession(guildID string, session *VoiceSession) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.sessions[guildID] = session
}

// oggPacketReader reads Ogg Opus pages and returns individual Opus packets,
// handling multi-packet pages and cross-page continuation.
type oggPacketReader struct {
	r       io.Reader
	buf     []byte  // partial packet accumulated across segments/pages
	isAudio bool    // true once granulePos > 0 has been seen

	// Remaining segments from the current page (lazy-read).
	segSizes []int
	segData  []byte
	segOff   int
	segGP    uint64
}

// next returns the next complete Opus packet. Header packets (OpusHead,
// OpusTags) are consumed automatically.
func (o *oggPacketReader) next() ([]byte, error) {
	for {
		// If we have no remaining segments, read the next Ogg page.
		if len(o.segSizes) == 0 {
			if err := o.readPage(); err != nil {
				return nil, err
			}
		}

		// Process one segment at a time.
		segSize := o.segSizes[0]
		o.segSizes = o.segSizes[1:]

		seg := o.segData[o.segOff : o.segOff+segSize]
		o.segOff += segSize

		o.buf = append(o.buf, seg...)

		if segSize < 255 {
			// End of a complete packet.
			pkt := o.buf
			o.buf = nil

			if o.segGP > 0 {
				o.isAudio = true
			}
			if o.isAudio {
				return pkt, nil
			}
			// Header packet — drop silently.
		}
	}
}

func (o *oggPacketReader) readPage() error {
	var header [27]byte
	if _, err := io.ReadFull(o.r, header[:]); err != nil {
		return err
	}
	if string(header[:4]) != "OggS" {
		return fmt.Errorf("invalid Ogg magic")
	}

	o.segGP = binary.LittleEndian.Uint64(header[6:14])

	segCount := header[26]
	segTable := make([]byte, segCount)
	if _, err := io.ReadFull(o.r, segTable); err != nil {
		return fmt.Errorf("segment table: %w", err)
	}

	totalSize := 0
	for _, s := range segTable {
		totalSize += int(s)
	}

	o.segData = make([]byte, totalSize)
	if _, err := io.ReadFull(o.r, o.segData); err != nil {
		return fmt.Errorf("page data: %w", err)
	}

	o.segOff = 0
	o.segSizes = make([]int, segCount)
	for i, s := range segTable {
		o.segSizes[i] = int(s)
	}

	return nil
}

func playAudioFile(vc *discordgo.VoiceConnection, fileURL string, stopChan chan struct{}) error {
	vc.Speaking(true)
	defer vc.Speaking(false)

	var inputPath string
	if fileURL != "" {
		if _, err := os.Stat(fileURL); err == nil {
			inputPath = fileURL
		} else {
			tmpFile, err := os.CreateTemp("", "audio-*")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			cmd := exec.Command("wget", "-q", "-O", tmpFile.Name(), fileURL)
			if err := cmd.Run(); err != nil {
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

	// FFmpeg: encode directly to Ogg Opus at 128kbps, 48kHz, mono
	ffmpegCmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:a", "libopus",
		"-b:a", "128k",
		"-ar", "48000",
		"-ac", "1",
		"-application", "audio",
		"-f", "opus",
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

	go io.Copy(io.Discard, stderr)

	opr := &oggPacketReader{r: stdout}

	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		pkt, err := opr.next()
		if err != nil {
			break
		}

		select {
		case vc.OpusSend <- pkt:
		case <-stopChan:
			return nil
		case <-time.After(time.Second):
			return fmt.Errorf("timed out sending opus frame")
		}
	}

	return nil
}

func (b *Bot) joinVoiceChannel(ctx context.Context, s *discordgo.Session, guildID, channelID string) (*discordgo.VoiceConnection, error) {
	// Leave any existing connection in this guild
	b.VoiceManager.StopSession(guildID)

	vc, err := s.ChannelVoiceJoin(ctx, guildID, channelID, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel: %w", err)
	}

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

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	vc, err := b.joinVoiceChannel(context.Background(), s, i.GuildID, channelID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				{
					Title:       "Error",
					Description: "Failed to join voice channel: " + err.Error(),
					Color:       0xf23f43,
				},
			},
		})
		return
	}

	b.VoiceManager.SetSession(i.GuildID, &VoiceSession{
		VoiceConnection: vc,
		GuildID:         i.GuildID,
		ChannelID:       channelID,
		PlaybackState:   PlaybackStopped,
		stopChan:        make(chan struct{}),
	})

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       "Joined Voice Channel",
				Description: "I've joined your voice channel. Use `/vc panel` to play recordings.",
				Color:       0x23a559,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: "https://i.ibb.co/67ZpGxTj/image.png",
				},
			},
		},
	})
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

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	b.VoiceManager.StopSession(i.GuildID)

	vc, err := b.joinVoiceChannel(context.Background(), s, i.GuildID, channelID)
	if err != nil {
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
			log.Printf("playback error: %v", playErr)
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
