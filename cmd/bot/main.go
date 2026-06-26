package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/spacebxr/o-8-discord-bot/internal/api"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
	"github.com/spacebxr/o-8-discord-bot/internal/discord"
	"github.com/spacebxr/o-8-discord-bot/internal/storage"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	token := os.Getenv("BOT_TOKEN")
	guildID := os.Getenv("GUILD_ID")
	roleHighCommand := os.Getenv("ROLE_HIGH_COMMAND")
	roleDevTeam := os.Getenv("ROLE_DEV_TEAM")
	discordClientID := os.Getenv("DISCORD_CLIENT_ID")
	discordClientSecret := os.Getenv("DISCORD_CLIENT_SECRET")
	discordRedirectURI := os.Getenv("DISCORD_REDIRECT_URI")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = api.GenerateJWTSecret()
		log.Println("JWT_SECRET not set, generated random secret")
	}

	if discordClientID == "" || discordClientSecret == "" || discordRedirectURI == "" {
		log.Fatal("Missing Discord OAuth environment variables (DISCORD_CLIENT_ID, DISCORD_CLIENT_SECRET, DISCORD_REDIRECT_URI)")
	}

	database, err := db.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Pool.Close()

	storageClient, err := storage.NewClient()
	if err != nil {
		log.Printf("Warning: S3 storage not configured: %v", err)
	}

	bot, err := discord.NewBot(token, database, storageClient, guildID, roleHighCommand, roleDevTeam)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	err = bot.Start()
	if err != nil {
		log.Fatal("Failed to start bot:", err)
	}
	defer bot.Stop()

	apiPort := os.Getenv("PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	apiServer := &api.Server{
		Database:           database,
		Session:            bot.Session,
		GuildID:            guildID,
		Storage:            storageClient,
		DiscordClientID:    discordClientID,
		DiscordClientSecret: discordClientSecret,
		DiscordRedirectURI: discordRedirectURI,
		JWTSecret:          jwtSecret,
	}
	go func() {
		fmt.Println("Starting API server on :" + apiPort)
		if err := apiServer.Start(apiPort); err != nil {
			log.Printf("API Server failed: %v", err)
		}
	}()

	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
