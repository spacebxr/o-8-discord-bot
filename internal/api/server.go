package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
)

type Server struct {
	Database *db.Database
	Session  *discordgo.Session
	GuildID  string
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
