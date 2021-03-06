package bot

import (
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AmputatorBot struct {
	DB         *gorm.DB
	DG         *discordgo.Session
	Config     AmputatorBotConfig
	StartingUp bool
}

type AmputatorBotConfig struct {
	AdminIds   []string `env:"ADMINISTRATOR_IDS"`
	DBHost     string   `env:"DB_HOST"`
	DBName     string   `env:"DB_NAME"`
	DBPassword string   `env:"DB_PASSWORD"`
	DBUser     string   `env:"DB_USER"`
	LogLevel   string   `env:"LOG_LEVEL"`
	Token      string   `env:"TOKEN"`
}

// BotReady is called when the bot is considered ready to use the Discord session.
func (bot *AmputatorBot) BotReady(s *discordgo.Session, r *discordgo.Ready) {
	for _, g := range r.Guilds {
		err := bot.registerOrUpdateGuild(s, g)
		if err != nil {
			log.Errorf("unable to register or update guild: %v", err)
		}
	}

	if bot.StartingUp {
		time.Sleep(time.Second * 10)
		bot.StartingUp = false
		err := bot.updateServersWatched(s)
		if err != nil {
			log.Error("unable to update servers watched")
		}
	}
}

// GuildCreate is called whenever the bot joins a new guild. It is also lazily called upon initial
// connection to Discord.
func (bot *AmputatorBot) GuildCreate(s *discordgo.Session, gc *discordgo.GuildCreate) {
	if gc.Guild.Unavailable {
		return
	}

	err := bot.registerOrUpdateGuild(s, gc.Guild)
	if err != nil {
		log.Errorf("unable to register or update guild: %v", err)
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func (bot *AmputatorBot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// This is a message the bot created itself
	if m.Author.ID == s.State.User.ID {
		bot.createMessageEvent("", m.Message)
		return
	}

	// Check if a message has the command prefix (global variable)
	if strings.HasPrefix(m.Content, commandPrefix) {
		var err error
		bot.createMessageEvent(statsCommand, m.Message)

		words := strings.Split(m.Content, " ")
		if len(words) < 2 {
			log.Warn("not enough words for ", statsCommand, " command")
			return
		}

		verb := words[1]
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
			log.Warn("problem handling ", configCommand, " command: ", err)
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
