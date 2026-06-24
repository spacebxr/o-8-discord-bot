package api

import (
	"context"
	"encoding/json"
	"net/http"

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
	Deployments   int64    `json:"deployments"`
	TotalMessages int64    `json:"totalMessages"`
	LastMessageAt string   `json:"lastMessageAt"`
	Strikes       []Strike `json:"strikes"`
}

func (s *Server) Start(port string) error {
	http.HandleFunc("/api/personnel", s.handleGetPersonnel)
	return http.ListenAndServe(":"+port, nil)
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
		if err == nil {
			username = user.Username
			if user.GlobalName != "" {
				username = user.GlobalName
			}
		}

		lastMsg := "Never"
		if stat.LastMessageAt != nil {
			lastMsg = stat.LastMessageAt.Format("2006-01-02 15:04 MST")
		}

		// Fetch strikes (infractions)
		infractions, _ := s.Database.GetInfractions(ctx, stat.UserID)
		var strikes []Strike
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
