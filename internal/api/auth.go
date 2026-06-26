package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type DiscordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

type AuthClaims struct {
	UserID   string   `json:"userId"`
	Username string   `json:"username"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

type AuthResponse struct {
	UserID   string   `json:"userId"`
	Username string   `json:"username"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
}

func GenerateJWTSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *Server) createJWT(userID, username, avatar string, roles []string) (string, error) {
	claims := AuthClaims{
		UserID:   userID,
		Username: username,
		Avatar:   avatar,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.JWTSecret))
}

func (s *Server) validateJWT(tokenStr string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AuthClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		claims, err := s.validateJWT(cookie.Value)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "claims", claims)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) handleAuthDiscord(w http.ResponseWriter, r *http.Request) {
	state := make([]byte, 16)
	rand.Read(state)
	stateStr := hex.EncodeToString(state)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    stateStr,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURL := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify&state=%s",
		s.DiscordClientID, s.DiscordRedirectURI, stateStr,
	)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (s *Server) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != state {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	token, err := s.exchangeCode(code)
	if err != nil {
		log.Printf("token exchange failed: %v", err)
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	discordUser, err := s.fetchDiscordUser(token)
	if err != nil {
		log.Printf("fetch user failed: %v", err)
		http.Error(w, "failed to fetch user", http.StatusInternalServerError)
		return
	}

	member, err := s.Session.GuildMember(s.GuildID, discordUser.ID)
	if err != nil {
		log.Printf("user %s not in guild: %v", discordUser.ID, err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<html><body><h1>Access Denied</h1><p>You must be a member of the Discord server to access this dashboard.</p></body></html>`))
		return
	}

	displayName := discordUser.Username
	if discordUser.GlobalName != "" {
		displayName = discordUser.GlobalName
	}
	avatarURL := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordUser.ID, discordUser.Avatar)
	if discordUser.Avatar == "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", (discordUser.ID[0]%5))
	}

	jwt, err := s.createJWT(discordUser.ID, displayName, avatarURL, member.Roles)
	if err != nil {
		log.Printf("jwt creation failed: %v", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    jwt,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authenticated":false}`))
		return
	}
	claims, err := s.validateJWT(cookie.Value)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authenticated":false}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"userId":        claims.UserID,
		"username":      claims.Username,
		"avatar":        claims.Avatar,
		"roles":         claims.Roles,
	})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) exchangeCode(code string) (string, error) {
	resp, err := http.PostForm("https://discord.com/api/oauth2/token", map[string][]string{
		"client_id":     {s.DiscordClientID},
		"client_secret": {s.DiscordClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.DiscordRedirectURI},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var data struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.Error != "" {
		return "", fmt.Errorf("discord error: %s", data.Error)
	}
	return data.AccessToken, nil
}

func (s *Server) fetchDiscordUser(accessToken string) (*DiscordUser, error) {
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}
