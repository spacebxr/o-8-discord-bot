package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/spacebxr/o-8-discord-bot/internal/db"
	"github.com/spacebxr/o-8-discord-bot/internal/discord"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	token := os.Getenv("BOT_TOKEN")
	guildID := os.Getenv("GUILD_ID")
	roleHighCommand := os.Getenv("ROLE_HIGH_COMMAND")

	if dbURL == "" || token == "" || guildID == "" || roleHighCommand == "" {
		log.Fatal("Missing required environment variables")
	}

	database, err := db.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Pool.Close()

	bot, err := discord.NewBot(token, database, guildID, roleHighCommand)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	err = bot.Start()
	if err != nil {
		log.Fatal("Failed to start bot:", err)
	}
	defer bot.Stop()

	fmt.Println("Bot is running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
