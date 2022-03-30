package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/go-sql-driver/mysql"
	cfg "github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

const (
	ampRegex string = ".*[./-]amp[-./]?.*"
)

var (
	config amputatorBotConfig
)

func init() {
	// Read from .env and override from the local environment
	dotEnvFeeder := feeder.DotEnv{Path: ".env"}
	envFeeder := feeder.Env{}
	cfg.New().AddFeeder(dotEnvFeeder).AddStruct(&config).Feed()
	cfg.New().AddFeeder(envFeeder).AddStruct(&config).Feed()

	logLevelSelection := log.InfoLevel
	switch {
	case strings.EqualFold(config.logLevel, "trace"):
		logLevelSelection = log.TraceLevel
		log.SetReportCaller(true)
	case strings.EqualFold(config.logLevel, "debug"):
		logLevelSelection = log.DebugLevel
		log.SetReportCaller(true)
	case strings.EqualFold(config.logLevel, "info"):
		logLevelSelection = log.InfoLevel
	case strings.EqualFold(config.logLevel, "warn"):
		logLevelSelection = log.WarnLevel
	case strings.EqualFold(config.logLevel, "error"):
		logLevelSelection = log.ErrorLevel
	}
	log.SetLevel(logLevelSelection)
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	// Set up DB connection as a property of a new amputatorBot
	dbConnected := true
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v", config.dbUser, config.dbPassword, config.dbHost, config.dbName)
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		dbConnected = false
	}
	defer db.Close()

	bot := amputatorBot{
		db:          db,
		dbConnected: dbConnected,
		dbUpdates:   make(chan string, 10),
		info:        botInfo{ID: config.botId},
		infoUpdates: make(chan botInfo, 10),
	}

	// create a new amputatorBot, with optional database connection
	bot.initializeStats()
	if err != nil {
		log.Error("unable to create new bot: ", err)
		os.Exit(1)
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.token)
	if err != nil {
		log.Error("error creating Discord session: ", err)
		os.Exit(1)
		return
	}

	// Start the stats handler
	go bot.statsHandler()
	defer close(bot.infoUpdates)

	// Start the db handler
	go bot.dbHandler()
	defer close(bot.dbUpdates)

	dg.AddHandler(bot.botReady)
	dg.AddHandler(bot.guildCreate)
	dg.AddHandler(bot.messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Error("error opening connection: ", err)
		os.Exit(1)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("bot started")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func (bot *amputatorBot) botReady(s *discordgo.Session, r *discordgo.Ready) {
	go bot.updateServersWatched(s, len(s.State.Guilds))
}

func (bot *amputatorBot) guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	go bot.updateServersWatched(s, len(s.State.Guilds))
}

func (bot *amputatorBot) statsHandler() {
	for stats := range bot.infoUpdates {
		bot.info = stats
	}
}

func (bot *amputatorBot) dbHandler() {
	for queryFragment := range bot.dbUpdates {
		err := bot.updateValueInDb(queryFragment)
		if err != nil {
			log.Error("error updating value in db: ", err)
		}
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (bot *amputatorBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	bot.updateMessagesSeen(bot.info.MessagesSeen + 1)

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if this is a direct message
	if m.GuildID == "" {
		if strings.HasPrefix(m.Content, "!stats") {
			log.Info("!stats called by ", m.Author.Username, "(", m.Author.ID, ")")
			bot.updateMessagesActedOn(bot.info.MessagesActedOn + 1)
			go bot.handleMessageWithStats(s, m)
			return
		}
	}

	// Check if the message has something that looks like an AMP URL according
	// to the ampRegex.
	obj, _ := regexp.Match(ampRegex, []byte(m.Content))
	if obj == true {
		log.Debug("message appears to have an AMP URL: ", m.Content)
		if config.automaticallyAmputate {
			bot.updateMessagesActedOn(bot.info.MessagesActedOn + 1)
			go bot.handleMessageWithAmpUrls(s, m)
			return
		} else {
			log.Info("URLs were not amputated because automatic amputation is not enabled")
			return
		}
	}
}
