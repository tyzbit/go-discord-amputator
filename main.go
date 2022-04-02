package main

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	cfg "github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type schemaTypes []interface{}

type amputatorBot struct {
	db *gorm.DB
	dg *discordgo.Session
	id string
}

type amputatorBotConfig struct {
	adminIds              []string `env:"ADMINISTRATOR_IDS"`
	automaticallyAmputate bool     `env:"AUTOMATICALLY_AMPUTATE"`
	botId                 int      `env:"BOT_ID"`
	dbHost                string   `env:"DB_HOST"`
	dbName                string   `env:"DB_NAME"`
	dbPassword            string   `env:"DB_PASSWORD"`
	dbUser                string   `env:"DB_USER"`
	guessAndCheck         string   `env:"GUESS_AND_CHECK"`
	logLevel              string   `env:"LOG_LEVEL"`
	maxDepth              string   `env:"MAX_DEPTH"`
	token                 string   `env:"TOKEN"`
}

const (
	ampRegex string = ".*[./-]amp[-./]?.*"
)

var (
	config         amputatorBotConfig
	allSchemaTypes = schemaTypes{
		&serverRegistration{},
		&serverConfig{},
		&urlInfo{},
		&amputationEvent{},
		&messageEvent{},
	}

	commandPrefix string = "!amp"
	statsCommand  string = "stats"
	configCommand string = "config"
)

// registerOrUpdateGuild checks if a guild is already registered in the database. If not,
// it creates it with sensibile defaults.
func (bot *amputatorBot) registerOrUpdateGuild(s *discordgo.Session, g *discordgo.Guild) {
	var registration serverRegistration
	bot.db.Find(&registration, g.ID)
	// TODO: There has to be a better way, but the guild name is blank in the s and g objects
	guild, err := s.Guild(g.ID)
	if err != nil {
		log.Error("unable to look up guild by id: ", g.ID)
	}
	// The server registration does not exist, so we will create with defaults
	if (registration == serverRegistration{}) {
		log.Info("creating registration for new server: ", guild.Name, "(", g.ID, ")")
		bot.db.Create(&serverRegistration{
			DiscordId: g.ID,
			Name:      guild.Name,
			UpdatedAt: time.Now(),
			Config:    defaultServerConfig,
		})
		return
	}

	err = bot.updateServersWatched(s)
	if err != nil {
		log.Error("unable to update servers watched: ", err)
	}
}

func init() {
	// Read from .env and override from the local environment
	dotEnvFeeder := feeder.DotEnv{Path: ".env"}
	envFeeder := feeder.Env{}
	_ = cfg.New().AddFeeder(dotEnvFeeder).AddStruct(&config).Feed()
	_ = cfg.New().AddFeeder(envFeeder).AddStruct(&config).Feed()

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
	var db *gorm.DB
	var err error
	var dbType string

	// Increase verbosity of the database if the loglevel is higher than Info
	var logConfig logger.Interface
	if log.GetLevel() > log.DebugLevel {
		logConfig = logger.Default.LogMode(logger.Info)
	}

	if config.dbHost != "" && config.dbName != "" && config.dbPassword != "" && config.dbUser != "" {
		dbType = "mysql"
		dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=True", config.dbUser, config.dbPassword, config.dbHost, config.dbName)
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logConfig})
	} else {
		dbType = "sqlite"
		db, err = gorm.Open(sqlite.Open("/var/go-discord-amputator/local.db"), &gorm.Config{Logger: logConfig})
	}
	if err != nil {
		log.Fatal("unable to connect to database (using "+dbType+"), err: ", err)
	}
	log.Info("using ", dbType)

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.token)
	if err != nil {
		log.Error("error creating Discord session: ", err)
		os.Exit(1)
		return
	}

	bot := amputatorBot{
		db: db,
		dg: dg,
		id: "",
	}

	// Set up DB if necessary
	for _, schemaType := range allSchemaTypes {
		err := db.AutoMigrate(schemaType)
		if err != nil {
			log.Fatal("unable to automigrate ", reflect.TypeOf(&schemaType).Elem().Name(), "err: ", err)
		}
	}

	dg.AddHandler(bot.botReady)
	dg.AddHandler(bot.guildCreate)
	dg.AddHandler(bot.messageCreate)

	discordIntents := discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages
	dg.Identify.Intents = discordIntents

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Error("error opening connection to discord: ", err)
		os.Exit(1)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("bot started")

	// Set our bot ID locally
	bot.id = dg.State.User.ID

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// botReady is called when the bot is considered ready to use the Discord session.
func (bot *amputatorBot) botReady(s *discordgo.Session, r *discordgo.Ready) {
	for _, g := range r.Guilds {
		bot.registerOrUpdateGuild(s, g)
	}
}

// guildCreate is called whenever the bot joins a new guild. It is also lazily called upon initial
// connection to Discord.
func (bot *amputatorBot) guildCreate(s *discordgo.Session, gc *discordgo.GuildCreate) {
	bot.registerOrUpdateGuild(s, gc.Guild)
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (bot *amputatorBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// This is a message the bot created itself
	if m.Author.ID == s.State.User.ID {
		bot.createMessageEvent("", m.Message)
		return
	}

	if strings.HasPrefix(m.Content, commandPrefix) {
		bot.createMessageEvent(statsCommand, m.Message)
		verb := strings.Split(m.Content, " ")[1]
		log.Info(verb+" called by ", m.Author.Username, "(", m.Author.ID, ")")
		switch verb {
		case statsCommand:
			err := bot.handleMessageWithStats(s, m)
			if err != nil {
				log.Warn("unable to handle ", statsCommand, " command: %w", err)
			}
			return
		case configCommand:
			err := bot.setServerConfig(s, m.Message)
			if err != nil {
				log.Warn("unable to handle ", configCommand, " command: %w", err)
			}
		default:
			log.Info("unknown command ", verb, " called")
			return
		}
	}

	// Check if the message has something that looks like an AMP URL according
	// to the ampRegex.
	match, _ := regexp.Match(ampRegex, []byte(m.Content))
	if match {
		bot.createMessageEvent("", m.Message)

		log.Debug("message appears to have an AMP URL: ", m.Content)
		err := bot.handleMessageWithAmpUrls(s, m)
		if err != nil {
			log.Warn("unable to handle message with AMP urls: %w", err)
		}
		return
	}
}
