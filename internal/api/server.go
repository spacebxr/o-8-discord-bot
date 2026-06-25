package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
	"github.com/spacebxr/o-8-discord-bot/internal/storage"
)

type Server struct {
	Database *db.Database
	Session  *discordgo.Session
	GuildID  string
	Storage *storage.Client
}

type Strike struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

type PersonnelResponse struct {
	ID            string   `json:"id"`
	Username      string   `json:"username"`
	AvatarURL     string   `json:"avatarUrl"`
	Deployments   int64    `json:"deployments"`
	TotalMessages int64    `json:"totalMessages"`
	LastMessageAt string   `json:"lastMessageAt"`
	Strikes       []Strike `json:"strikes"`
}

func (s *Server) Start(port string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/personnel", s.handleGetPersonnel)
	mux.HandleFunc("/api/role-connections", s.handleRoleConnections)
	mux.HandleFunc("/api/roles", s.handleGetRoles)
	mux.HandleFunc("/api/announce/deployment", s.handleAnnounceDeployment)
	mux.HandleFunc("/api/deployments", s.handleDeployments)
	mux.HandleFunc("/api/deployments/start", s.handleDeployStart)
	mux.HandleFunc("/api/deployments/ongoing", s.handleDeployOngoing)
	mux.HandleFunc("/api/deployments/end", s.handleDeployEnd)
	mux.HandleFunc("/api/recordings", s.handleRecordings)
	mux.HandleFunc("/api/recordings/upload", s.handleRecordingUpload)
	mux.HandleFunc("/api/recordings/delete", s.handleRecordingDelete)

	distPath := "dashboard/dist"
	if _, err := os.Stat(distPath); err != nil {
		if _, errOpt := os.Stat("/dashboard/dist"); errOpt == nil {
			distPath = "/dashboard/dist"
		}
	}

	if _, err := os.Stat(distPath); err == nil {
		log.Printf("Serving static dashboard files from: %s", distPath)
		fs := http.FileServer(http.Dir(distPath))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			filePath := distPath + r.URL.Path
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				http.ServeFile(w, r, distPath+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	} else {
		log.Printf("Warning: dashboard/dist or /dashboard/dist not found. Dashboard serving is disabled.")
	}

	return http.ListenAndServe(":"+port, mux)
}

func (s *Server) handleGetPersonnel(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := context.Background()
	stats, err := s.Database.GetPersonnelStats(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch roles to resolve mentions
	roles, err := s.Session.GuildRoles(s.GuildID)
	roleMap := make(map[string]*discordgo.Role)
	if err == nil {
		for _, r := range roles {
			roleMap[r.ID] = r
		}
	}
	mentionRegex := regexp.MustCompile(`<@&(\d+)>`)

	var response []PersonnelResponse
	for _, stat := range stats {
		user, err := s.Session.User(stat.UserID)
		username := "Unknown"
		avatarUrl := ""
		if err == nil {
			username = user.Username
			if user.GlobalName != "" {
				username = user.GlobalName
			}
			avatarUrl = user.AvatarURL("")
		}

		lastMsg := "Never"
		if stat.LastMessageAt != nil {
			lastMsg = stat.LastMessageAt.Format("2006-01-02 15:04 MST")
		}

		infractions, _ := s.Database.GetInfractions(ctx, stat.UserID)
		strikes := []Strike{}
		for _, inf := range infractions {
			reason := inf.Reason
			if inf.Punishment != "Strike" && inf.Punishment != "strike" {
				reason = inf.Punishment + " - " + reason
			}

			// Replace role mentions with HTML
			reason = mentionRegex.ReplaceAllStringFunc(reason, func(m string) string {
				roleID := mentionRegex.FindStringSubmatch(m)[1]
				if role, exists := roleMap[roleID]; exists {
					color := "var(--text-primary)"
					if role.Color != 0 {
						color = fmt.Sprintf("#%06x", role.Color)
					}
					return fmt.Sprintf(`<span style="color: %s; background-color: color-mix(in srgb, %s 15%%, transparent); padding: 2px 6px; border-radius: 4px; font-weight: 500;">@%s</span>`, color, color, role.Name)
				}
				return m
			})

			strikes = append(strikes, Strike{
				ID:     inf.ID,
				Reason: reason,
				Date:   inf.CreatedAt.Format("2006-01-02"),
			})
		}

		response = append(response, PersonnelResponse{
			ID:            stat.UserID,
			Username:      username,
			AvatarURL:     avatarUrl,
			Deployments:   stat.DeploymentsParticipated,
			TotalMessages: stat.TotalMessages,
			LastMessageAt: lastMsg,
			Strikes:       strikes,
		})
	}


	if response == nil {
		response = []PersonnelResponse{}
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleRoleConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ctx := context.Background()

	switch r.Method {
	case "GET":
		connections, err := s.Database.GetRoleConnections(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		roles, _ := s.Session.GuildRoles(s.GuildID)
		roleMap := make(map[string]*discordgo.Role)
		for _, role := range roles {
			roleMap[role.ID] = role
		}
		type EnrichedConnection struct {
			ID        string `json:"id"`
			RoleIDA   string `json:"roleIdA"`
			RoleNameA string `json:"roleNameA"`
			ColorA    string `json:"colorA"`
			RoleIDB   string `json:"roleIdB"`
			RoleNameB string `json:"roleNameB"`
			ColorB    string `json:"colorB"`
		}
		result := make([]EnrichedConnection, 0, len(connections))
		for _, c := range connections {
			ec := EnrichedConnection{ID: c.ID, RoleIDA: c.RoleIDA, RoleIDB: c.RoleIDB}
			if rA, ok := roleMap[c.RoleIDA]; ok {
				ec.RoleNameA = rA.Name
				if rA.Color != 0 {
					ec.ColorA = fmt.Sprintf("#%06x", rA.Color)
				}
			}
			if rB, ok := roleMap[c.RoleIDB]; ok {
				ec.RoleNameB = rB.Name
				if rB.Color != 0 {
					ec.ColorB = fmt.Sprintf("#%06x", rB.Color)
				}
			}
			result = append(result, ec)
		}
		json.NewEncoder(w).Encode(result)

	case "POST":
		var body struct {
			RoleIDA string `json:"roleIdA"`
			RoleIDB string `json:"roleIdB"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RoleIDA == "" || body.RoleIDB == "" {
			http.Error(w, "invalid body: roleIdA and roleIdB required", http.StatusBadRequest)
			return
		}
		id, err := s.Database.AddRoleConnection(ctx, body.RoleIDA, body.RoleIDB)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"id": id})

	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id query param required", http.StatusBadRequest)
			return
		}
		if err := s.Database.DeleteRoleConnection(ctx, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetRoles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	roles, err := s.Session.GuildRoles(s.GuildID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type RoleItem struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Color string `json:"color"`
	}

	result := make([]RoleItem, 0, len(roles))
	for _, r := range roles {
		color := ""
		if r.Color != 0 {
			color = fmt.Sprintf("#%06x", r.Color)
		}
		result = append(result, RoleItem{ID: r.ID, Name: r.Name, Color: color})
	}
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleAnnounceDeployment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Message  string `json:"message"`
		HostID   string `json:"hostId"`
		CoHostID string `json:"coHostId"`
		Location string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	hostMention := "Unknown"
	if body.HostID != "" {
		hostMention = "<@" + body.HostID + ">"
	}
	cohostMention := "Unknown"
	if body.CoHostID != "" {
		cohostMention = "<@" + body.CoHostID + ">"
	}

	description := fmt.Sprintf(
		"%s\n\n**Host:** %s\n**Co-Host:** %s\n**Participants:** None\n**Location:** %s",
		body.Message, hostMention, cohostMention, body.Location,
	)

	var pings []string
	if body.HostID != "" {
		pings = append(pings, "<@"+body.HostID+">")
	}
	if body.CoHostID != "" {
		pings = append(pings, "<@"+body.CoHostID+">")
	}
	pingsStr := ""
	if len(pings) > 0 {
		for i, p := range pings {
			if i > 0 {
				pingsStr += " "
			}
			pingsStr += p
		}
	}

	channels, err := s.Session.GuildChannels(s.GuildID)
	if err != nil {
		http.Error(w, "failed to fetch channels: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var deployChannelID string
	for _, ch := range channels {
		if ch.Name == "💻‖deployments" {
			deployChannelID = ch.ID
			break
		}
	}
	if deployChannelID == "" {
		http.Error(w, "could not find 💻‖deployments channel", http.StatusNotFound)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Deployment Scheduled",
		Description: description,
		Color:       0xFAA61A,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://i.ibb.co/67ZpGxTj/image.png",
		},
	}

	msg, err := s.Session.ChannelMessageSendComplex(deployChannelID, &discordgo.MessageSend{
		Content: pingsStr,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		http.Error(w, "failed to send message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: deployChannelID,
		ID:      msg.ID,
		Content: &pingsStr,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
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
	s.Session.MessageReactionAdd(deployChannelID, msg.ID, "✅")

	_, _ = s.Database.CreateDeployment(context.Background(), body.Message, body.HostID, body.CoHostID, body.Location, msg.ID, "dashboard")

	json.NewEncoder(w).Encode(map[string]string{"messageId": msg.ID})
}

func (s *Server) handleDeployments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deployments, err := s.Database.GetDeployments(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enrich with usernames
	type DeployResp struct {
		db.Deployment
		HostName    string `json:"hostName"`
		CoHostName  string `json:"coHostName"`
		ParticipantCount int `json:"participantCount"`
	}

	roleMap := make(map[string]*discordgo.Role)
	roles, err := s.Session.GuildRoles(s.GuildID)
	if err == nil {
		for _, ro := range roles {
			roleMap[ro.ID] = ro
		}
	}

	resp := make([]DeployResp, 0, len(deployments))
	for _, d := range deployments {
		hostName := "Unknown"
		if d.HostID != "" {
			if u, err := s.Session.User(d.HostID); err == nil {
				hostName = u.Username
				if u.GlobalName != "" {
					hostName = u.GlobalName
				}
			}
		}
		coHostName := "Unknown"
		if d.CoHostID != "" {
			if u, err := s.Session.User(d.CoHostID); err == nil {
				coHostName = u.Username
				if u.GlobalName != "" {
					coHostName = u.GlobalName
				}
			}
		}

		participants, _ := s.Database.GetDeploymentParticipants(context.Background(), d.DiscordMessageID)

		resp = append(resp, DeployResp{
			Deployment:       d,
			HostName:         hostName,
			CoHostName:       coHostName,
			ParticipantCount: len(participants),
		})
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDeployStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MessageID == "" {
		http.Error(w, "messageId required", http.StatusBadRequest)
		return
	}

	channels, err := s.Session.GuildChannels(s.GuildID)
	if err != nil {
		http.Error(w, "failed to fetch channels", http.StatusInternalServerError)
		return
	}
	var deployChannelID string
	for _, ch := range channels {
		if ch.Name == "💻‖deployments" {
			deployChannelID = ch.ID
			break
		}
	}
	if deployChannelID == "" {
		http.Error(w, "could not find 💻‖deployments channel", http.StatusNotFound)
		return
	}

	msg, err := s.Session.ChannelMessage(deployChannelID, body.MessageID)
	if err != nil {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	if len(msg.Embeds) == 0 {
		http.Error(w, "no embed found in message", http.StatusBadRequest)
		return
	}

	embed := msg.Embeds[0]
	embed.Title = "Deployment Started"
	embed.Color = 0x23a559
	startTime := time.Now().UTC()
	embed.Description += fmt.Sprintf("\n\n**Started:** <t:%d:F>", startTime.Unix())

	s.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: deployChannelID,
		ID:      body.MessageID,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Status: Ongoing",
						Style:    discordgo.PrimaryButton,
						CustomID: "deploy_ongoing_" + body.MessageID,
					},
				},
			},
		},
	})

	_ = s.Database.StartDeployment(context.Background(), body.MessageID, startTime)

	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (s *Server) handleDeployOngoing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MessageID == "" {
		http.Error(w, "messageId required", http.StatusBadRequest)
		return
	}

	channels, err := s.Session.GuildChannels(s.GuildID)
	if err != nil {
		http.Error(w, "failed to fetch channels", http.StatusInternalServerError)
		return
	}
	var deployChannelID string
	for _, ch := range channels {
		if ch.Name == "💻‖deployments" {
			deployChannelID = ch.ID
			break
		}
	}
	if deployChannelID == "" {
		http.Error(w, "could not find 💻‖deployments channel", http.StatusNotFound)
		return
	}

	msg, err := s.Session.ChannelMessage(deployChannelID, body.MessageID)
	if err != nil {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	if len(msg.Embeds) == 0 {
		http.Error(w, "no embed found in message", http.StatusBadRequest)
		return
	}

	embed := msg.Embeds[0]
	embed.Title = "Deployment Ongoing"
	embed.Color = 0x5865F2

	s.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel: deployChannelID,
		ID:      body.MessageID,
		Embeds:  &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "End Deployment",
						Style:    discordgo.DangerButton,
						CustomID: "deploy_end_" + body.MessageID,
					},
				},
			},
		},
	})

	_ = s.Database.UpdateDeploymentStatus(context.Background(), body.MessageID, "ongoing")

	json.NewEncoder(w).Encode(map[string]string{"status": "ongoing"})
}

func (s *Server) handleDeployEnd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		MessageID string `json:"messageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MessageID == "" {
		http.Error(w, "messageId required", http.StatusBadRequest)
		return
	}

	channels, err := s.Session.GuildChannels(s.GuildID)
	if err != nil {
		http.Error(w, "failed to fetch channels", http.StatusInternalServerError)
		return
	}
	var deployChannelID string
	var logChannelID string
	for _, ch := range channels {
		if ch.Name == "💻‖deployments" {
			deployChannelID = ch.ID
		}
		if ch.Name == "💻‖deployments-logs" {
			logChannelID = ch.ID
		}
	}
	if deployChannelID == "" {
		http.Error(w, "could not find 💻‖deployments channel", http.StatusNotFound)
		return
	}

	msg, err := s.Session.ChannelMessage(deployChannelID, body.MessageID)
	if err != nil {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	if len(msg.Embeds) == 0 {
		http.Error(w, "no embed found in message", http.StatusBadRequest)
		return
	}

	endTime := time.Now().UTC()
	dep, _ := s.Database.GetDeploymentByMessageID(context.Background(), body.MessageID)
	var durationStr string
	var durationSec int64
	if dep != nil && dep.StartedAt != nil {
		duration := endTime.Sub(*dep.StartedAt)
		durationSec = int64(duration.Seconds())
		h := int(duration.Hours())
		m := int(duration.Minutes()) % 60
		sec := int(duration.Seconds()) % 60
		durationStr = fmt.Sprintf("%02dh %02dm %02ds", h, m, sec)
	} else {
		durationStr = "Unknown"
	}

	embed := msg.Embeds[0]
	embed.Title = "Deployment Ended"
	embed.Color = 0xf23f43

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

	s.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    deployChannelID,
		ID:         body.MessageID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{},
	})

	_ = s.Database.EndDeployment(context.Background(), body.MessageID, endTime, durationSec)

	if logChannelID != "" {
		s.Session.ChannelMessageSendEmbed(logChannelID, &discordgo.MessageEmbed{
			Title:       "Deployment Report 📋",
			Description: embed.Description,
			Color:       0xf23f43,
			Fields:      embed.Fields,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: "https://i.ibb.co/67ZpGxTj/image.png",
			},
		})
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ended"})
}

func (s *Server) handleRecordings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recordings, err := s.Database.GetAudioRecordings(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type recordingResponse struct {
		ID              string  `json:"id"`
		Name            string  `json:"name"`
		FileURL         string  `json:"fileUrl"`
		DurationSeconds float64 `json:"durationSeconds"`
		CreatedAt       string  `json:"createdAt"`
	}

	resp := make([]recordingResponse, 0, len(recordings))
	for _, rec := range recordings {
		fileURL := rec.FileURL
		if s.Storage != nil {
			if u, err := s.Storage.GetURL(context.Background(), rec.FileURL); err == nil {
				fileURL = u
			}
		}
		resp = append(resp, recordingResponse{
			ID:              rec.ID,
			Name:            rec.Name,
			FileURL:         fileURL,
			DurationSeconds: rec.DurationSeconds,
			CreatedAt:       rec.CreatedAt.Format(time.RFC3339),
		})
	}

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleRecordingUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Storage == nil {
		http.Error(w, "Storage not configured", http.StatusInternalServerError)
		return
	}

	err := r.ParseMultipartForm(50 << 20)
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	objectName := fmt.Sprintf("recordings/%s", header.Filename)
	objectKey, err := s.Storage.Upload(context.Background(), objectName, header.Header.Get("Content-Type"), file, header.Size)
	if err != nil {
		http.Error(w, "failed to upload file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = s.Database.AddAudioRecording(context.Background(), name, objectKey, 0, "dashboard")
	if err != nil {
		http.Error(w, "failed to save recording: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleRecordingDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	err := s.Database.DeleteAudioRecording(context.Background(), body.ID)
	if err != nil {
		http.Error(w, "failed to delete recording: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

