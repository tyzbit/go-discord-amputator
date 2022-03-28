package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const (
	ampRegex string = ".*[./-]amp[-./]?.*"
)

// Environment variable name definitions
const (
	adminIds              string = "ADMINISTRATOR_IDS"
	automaticallyAmputate string = "AUTOMATICALLY_AMPUTATE"
	botId                 string = "BOT_ID"
	dbHost                string = "DB_HOST"
	dbName                string = "DB_NAME"
	dbPassword            string = "DB_PASSWORD"
	dbUser                string = "DB_USER"
	guessAndCheck         string = "GUESS_AND_CHECK"
	logLevel              string = "LOG_LEVEL"
	maxDepth              string = "MAX_DEPTH"
	token                 string = "TOKEN"
)

var (
	env   map[string]string
	stats = map[string]int{
		"messagesSeen":        0,
		"messagesActedOn":     0,
		"messagesSent":        0,
		"callsToAmputatorApi": 0,
		"urlsAmputated":       0,
		"serversWatched":      0,
	}
)

func init() {
	// Read from .env first
	env, _ = godotenv.Read(".env")

	// Override with values from environment
	for _, envDeclaration := range os.Environ() {
		parsedDeclaration := strings.SplitN(envDeclaration, "=", 2)
		env[parsedDeclaration[0]] = parsedDeclaration[1]
	}

	logLevelSelection := logrus.InfoLevel
	switch {
	case strings.EqualFold(env[logLevel], "debug"):
		logLevelSelection = logrus.DebugLevel
	case strings.EqualFold(env[logLevel], "info"):
		logLevelSelection = logrus.DebugLevel
	case strings.EqualFold(env[logLevel], "warn"):
		logLevelSelection = logrus.WarnLevel
	case strings.EqualFold(env[logLevel], "error"):
		logLevelSelection = logrus.ErrorLevel
	}
	logrus.SetLevel(logLevelSelection)
}

func main() {
	dbConnected := true
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v", env[dbUser], env[dbPassword], env[dbHost], env[dbName])
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		dbConnected = false
	}
	defer db.Close()

	id, err := strconv.Atoi(env[botId])
	if err != nil {
		id = 1
	}
	bot := amputatorBot{
		dbConnected:  dbConnected,
		dbConnection: db,
		id:           id,
		stats:        stats,
		updateStats:  make(chan map[string]int, 10),
	}
	bot, botError := bot.updateOrInitializeBotStats()
	if botError != nil {
		logrus.Warn("unable to set up stats: ", botError)
		bot.dbConnected = false
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + env[token])
	if err != nil {
		logrus.Error("error creating Discord session: ", err)
		os.Exit(1)
		return
	}

	// Start the stats handler
	go bot.statsHandler()
	defer close(bot.updateStats)

	dg.AddHandler(bot.botReady)
	dg.AddHandler(bot.guildCreate)
	dg.AddHandler(bot.messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		logrus.Error("error opening connection: ", err)
		os.Exit(1)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	logrus.Info("bot started")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func (bot amputatorBot) botReady(s *discordgo.Session, r *discordgo.Ready) {
	go bot.updateServersWatched(s, len(s.State.Guilds))
}

func (bot amputatorBot) guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	go bot.updateServersWatched(s, len(s.State.Guilds))
}

func (bot amputatorBot) statsHandler() {
	for stats := range bot.updateStats {
		for stat, value := range stats {
			bot.stats[stat] = value
			err := bot.writeStatToDatabase(stat, value)
			if err != nil {
				logrus.Error("unable to write stat to database: ", err)
			}
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (bot amputatorBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	bot.updateStats <- map[string]int{"messagesSeen": bot.stats["messagesSeen"] + 1}

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if this is a direct message
	if m.GuildID == "" {
		if strings.HasPrefix(m.Content, "!stats") {
			logrus.Debug("!stats called by ", m.Author.Username, "(", m.Author.ID, ")")
			bot.updateStats <- map[string]int{"messagesActedOn": bot.stats["messagesActedOn"] + 1}
			go bot.handleMessageWithStats(s, m)
			return
		}
	}

	// Check if the message has something that looks like an AMP URL according
	// to the ampRegex.
	obj, _ := regexp.Match(ampRegex, []byte(m.Content))
	if obj == true {
		logrus.Debug("message appears to have an AMP URL")
		if env[automaticallyAmputate] != "" {
			bot.updateStats <- map[string]int{"messagesActedOn": bot.stats["messagesActedOn"] + 1}
			go bot.handleMessageWithAmpUrls(s, m)
			return
		} else {
			logrus.Info("URLs were not amputated because ", automaticallyAmputate, " was not set")
			return
		}
	}
}
