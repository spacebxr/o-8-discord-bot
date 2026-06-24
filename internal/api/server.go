package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
)

type Server struct {
	Database *db.Database
	Session  *discordgo.Session
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
			if inf.Punishment == "Strike" || inf.Punishment == "strike" {
				strikes = append(strikes, Strike{
					ID:     inf.ID,
					Reason: inf.Reason,
					Date:   inf.CreatedAt.Format("2006-01-02"),
				})
			} else {
				strikes = append(strikes, Strike{
					ID:     inf.ID,
					Reason: inf.Punishment + " - " + inf.Reason,
					Date:   inf.CreatedAt.Format("2006-01-02"),
				})
			}
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
