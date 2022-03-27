package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/mvdan/xurls"
	"github.com/sirupsen/logrus"
	goamputate "github.com/tyzbit/go-amputate"
)

const (
	userAgent    string = "Discord-Amputator bot"
	ampRegex     string = ".*[./-]amp[-./]?.*"
	amputatorApi string = "https://www.amputatorbot.com/api/v1/"
)

// Environment variable name definitions
const (
	adminIds              string = "ADMINISTRATOR_IDS"
	automaticallyAmputate string = "AUTOMATICALLY_AMPUTATE"
	guessAndCheck         string = "GUESS_AND_CHECK"
	logLevel              string = "LOG_LEVEL"
	maxDepth              string = "MAX_DEPTH"
	token                 string = "TOKEN"
)

const (
	messagesSeen        string = "Messages Seen"
	messagesActedOn     string = "Messages Acted On"
	messagesSent        string = "Messages Sent"
	callsToAmputatorApi string = "Calls to Amputator API"
	urlsAmputated       string = "URLs Amputated"
	serversWatched      string = "Servers Watched"
)

var (
	env   map[string]string
	stats map[string]int = map[string]int{
		messagesSeen:        0,
		messagesActedOn:     0,
		messagesSent:        0,
		callsToAmputatorApi: 0,
		urlsAmputated:       0,
		serversWatched:      0,
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
}

func _initLogging() {
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
	_initLogging()

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + env[token])
	if err != nil {
		logrus.Error("error creating Discord session: ", err)
		os.Exit(1)
		return
	}

	dg.AddHandler(botReady)
	dg.AddHandler(guildCreate)
	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		logrus.Error("error opening connection: ", err)
		os.Exit(1)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	logrus.Info("Bot started")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func botReady(s *discordgo.Session, r *discordgo.Ready) {
	stats[serversWatched] = len(s.State.Guilds)
	updateServersWatched(s, stats[serversWatched])
}

func guildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	stats[serversWatched] = len(s.State.Guilds)
	updateServersWatched(s, stats[serversWatched])
}

func updateServersWatched(s *discordgo.Session, serverCount int) {
	logrus.Info("watching ", serverCount, " servers")
	usd := &discordgo.UpdateStatusData{Status: "online"}
	usd.Activities = make([]*discordgo.Activity, 1)
	usd.Activities[0] = &discordgo.Activity{
		Name: fmt.Sprintf("%v servers", serverCount),
		Type: discordgo.ActivityTypeWatching,
		URL:  "https://github.com/tyzbit/go-discord-amputator",
	}

	err := s.UpdateStatusComplex(*usd)
	if err != nil {
		logrus.Error("failed to set status: %v", err)
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	stats[messagesSeen]++

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if this is a direct message
	if m.GuildID == "" {
		if strings.HasPrefix(m.Content, "!stats") {
			logrus.Debug("!stats called by ", m.Author.Username, " id: ", m.Author.ID)
			stats[messagesActedOn]++
			go handleMessageWithStats(s, m)
			return
		}
	}

	// Check if the message has something that looks like an AMP URL according
	// to the ampRegex.
	obj, _ := regexp.Match(ampRegex, []byte(m.Content))
	if obj == true {
		logrus.Debug("message appears to have an AMP URL")
		if env[automaticallyAmputate] != "" {
			stats[messagesActedOn]++
			go handleMessageWithAmpUrls(s, m)
			return
		} else {
			logrus.Info("URLs were not amputated because ", automaticallyAmputate, " was not set")
			return
		}
	}
}

// handleMessageWithStats takes a discord session and a user ID and sends a
// message to the user with stats about the bot.
func handleMessageWithStats(s *discordgo.Session, m *discordgo.MessageCreate) {
	administrator := false
	for _, id := range strings.Split(env[adminIds], ",") {
		if m.Author.ID == id {
			administrator = true
		}
	}

	if administrator {
		formattedStats := ""
		for stat, value := range stats {
			formattedStats = fmt.Sprintf("%v\n%v: %v", formattedStats, stat, value)
		}

		stats[messagesSent]++
		embed := &discordgo.MessageEmbed{
			Title:       "Amputation Stats",
			Description: formattedStats,
		}

		logrus.Debug("sending !stats response to ", m.Author.Username, "(", m.Author.ID, ")")
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
	} else {
		logrus.Debug("did not respond to ", m.Author.Username, " id ", m.Author.ID, " because user is not an administrator")
	}
}

// handleMessageWithAmpUrls takes a Discord session and a message string and
// calls go-amputator with a []string of URLs parsed from the message
// It then sends an embed with the resulting amputated URLs.
func handleMessageWithAmpUrls(s *discordgo.Session, m *discordgo.MessageCreate) {
	xurlsRelaxed := xurls.Strict
	urls := xurlsRelaxed.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		logrus.Debug("found 0 URLs in message that matched amp regex: ", ampRegex)
		return
	}

	logrus.Debug("URLs parsed from message: ", strings.Join(urls, ", "))

	var bot goamputate.AmputatorBot
	options := map[string]string{}

	// Read environment options and set parameters appropriately
	if env[guessAndCheck] != "" {
		options["gac"] = env[guessAndCheck]
	}
	if env[maxDepth] != "" {
		options["md"] = env[maxDepth]
	}

	genericLinkAmputationFailureMessage := &discordgo.MessageEmbed{
		Title:       "Problem Amputating",
		Description: "Sorry, I couldn't amputate that link.",
	}
	stats[callsToAmputatorApi]++
	amputatedLinks, err := bot.Amputate(urls, options)
	if err != nil {
		logrus.Error("error calling Amputator API: ", err)
		stats[messagesSent]++
		s.ChannelMessageSendEmbed(m.ChannelID, genericLinkAmputationFailureMessage)
		return
	}

	if len(amputatedLinks) == 0 {
		logrus.Warn("amputator bot returned no Amputated URLs from: ", strings.Join(urls, ", "))
		stats[messagesSent]++
		s.ChannelMessageSendEmbed(m.ChannelID, genericLinkAmputationFailureMessage)
		return
	}
	stats[urlsAmputated] = stats[urlsAmputated] + len(amputatedLinks)

	plural := ""
	if len(amputatedLinks) > 1 {
		plural = "s"
	}

	title := fmt.Sprintf("Amputated Link%v", plural)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: strings.Join(amputatedLinks, "\n"),
	}
	stats[messagesSent]++
	guild, err := s.Guild(m.GuildID)
	guildName := "unknown"
	if err != nil {
		logrus.Warn("couldn't get guild for ID ", m.GuildID)
	}
	if guild.Name != "" {
		guildName = guild.Name
	}
	logrus.Debug("sending amputate message response in ", guildName, "(", m.GuildID, "), calling user: ", m.Author.Username, "(", m.Author.ID, ")")
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
