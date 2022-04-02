package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	cfg "github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type amputatorBot struct {
	db *gorm.DB
	dg *discordgo.Session
}

type amputatorBotConfig struct {
	adminIds   []string `env:"ADMINISTRATOR_IDS"`
	dbHost     string   `env:"DB_HOST"`
	dbName     string   `env:"DB_NAME"`
	dbPassword string   `env:"DB_PASSWORD"`
	dbUser     string   `env:"DB_USER"`
	logLevel   string   `env:"LOG_LEVEL"`
	token      string   `env:"TOKEN"`
}

const (
	ampRegex string = ".*[./-]amp[-./]?.*"

	sqlitePath string = "/var/go-discord-amputator/local.db"

	commandPrefix string = "!amp"
	statsCommand  string = "stats"
	configCommand string = "config"
)

var (
	config         amputatorBotConfig
	allSchemaTypes = []interface{}{
		&serverRegistration{},
		&serverConfig{},
		&urlInfo{},
		&amputationEvent{},
		&messageEvent{},
	}
)

func init() {
	// Read from .env and override from the local environment
	dotEnvFeeder := feeder.DotEnv{Path: ".env"}
	envFeeder := feeder.Env{}

	_ = cfg.New().AddFeeder(dotEnvFeeder).AddStruct(&config).Feed()
	_ = cfg.New().AddFeeder(envFeeder).AddStruct(&config).Feed()

	// Info level by default
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
		// Create the folder path if it doesn't exist
		_, err = os.Stat(sqlitePath)
		if errors.Is(err, fs.ErrNotExist) {
			dirPath := filepath.Dir(sqlitePath)
			if err := os.MkdirAll(dirPath, 0660); err != nil {
				log.Fatal("unable to make directory path ", dirPath, " err: ", err)
			}
		}
		db, err = gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{Logger: logConfig})
	}

	if err != nil {
		log.Fatal("unable to connect to database (using "+dbType+"), err: ", err)
	}

	log.Info("using ", dbType, " for the database")

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + config.token)
	if err != nil {
		log.Fatal("error creating Discord session: ", err)
	}

	// amputatorBot is an instance of this bot. It has many methods attached to
	// it for controlling the bot. db is the database object, dg is the
	// discordgo object.
	bot := amputatorBot{
		db: db,
		dg: dg,
	}

	// Set up DB if necessary
	for _, schemaType := range allSchemaTypes {
		err := db.AutoMigrate(schemaType)
		if err != nil {
			log.Fatal("unable to automigrate ", reflect.TypeOf(&schemaType).Elem().Name(), "err: ", err)
		}
	}

	// These handlers get called whenever there's a corresponding
	// Discord event.
	dg.AddHandler(bot.botReady)
	dg.AddHandler(bot.guildCreate)
	dg.AddHandler(bot.messageCreate)

	// We have to be explicit about what we want to receive. In addition,
	// some intents require additional permissions, which must be granted
	// to the bot when it's added or after the fact by a guild admin.
	discordIntents := discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages
	dg.Identify.Intents = discordIntents

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Fatal("error opening connection to discord: ", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("bot started")

	// Listen for signals from the OS
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// botReady is called when the bot is considered ready to use the Discord session.
func (bot *amputatorBot) botReady(s *discordgo.Session, r *discordgo.Ready) {
	for _, g := range r.Guilds {
		err := bot.registerOrUpdateGuild(s, g)
		if err != nil {
			log.Errorf("unable to register or update guild: %v", err)
		}
	}
}

// guildCreate is called whenever the bot joins a new guild. It is also lazily called upon initial
// connection to Discord.
func (bot *amputatorBot) guildCreate(s *discordgo.Session, gc *discordgo.GuildCreate) {
	err := bot.registerOrUpdateGuild(s, gc.Guild)
	if err != nil {
		log.Errorf("unable to register or update guild: %v", err)
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (bot *amputatorBot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// This is a message the bot created itself
	if m.Author.ID == s.State.User.ID {
		bot.createMessageEvent("", m.Message)
		return
	}

	// Check if a message has the command prefix (global variable)
	if strings.HasPrefix(m.Content, commandPrefix) {
		var err error
		bot.createMessageEvent(statsCommand, m.Message)
		verb := strings.Split(m.Content, " ")[1]
		log.Info(verb+" called by ", m.Author.Username, "(", m.Author.ID, ")")
		switch verb {
		case statsCommand:
			err = bot.handleMessageWithStats(s, m)
		case configCommand:
			err = bot.setServerConfig(s, m.Message)
		default:
			log.Warn("unknown command ", verb, " called")
		}

		if err != nil {
			log.Warn("problem handling ", configCommand, " command: %w", err)
		}
		return
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
