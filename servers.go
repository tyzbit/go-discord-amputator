package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type serverRegistration struct {
	DiscordId string `gorm:"primaryKey"`
	Name      string
	UpdatedAt time.Time
	Config    serverConfig `gorm:"foreignKey:DiscordId"`
}

type serverConfig struct {
	DiscordId              string `gorm:"primaryKey"`
	Name                   string
	AmputationEnabled      bool
	ReplyToOriginalMessage bool
	UseEmbed               bool
	AllowStats             bool
	GuessAndCheck          bool
	MaxDepth               int
}

var defaultServerConfig serverConfig = serverConfig{
	DiscordId:              "0",
	Name:                   "default",
	AmputationEnabled:      true,
	ReplyToOriginalMessage: false,
	UseEmbed:               true,
	AllowStats:             true,
	GuessAndCheck:          true,
	MaxDepth:               3,
}

// getServerConfig takes a guild ID and returns a serverConfig object for that server
func (bot *amputatorBot) getServerConfig(guildId string) serverConfig {
	sc := serverConfig{}
	bot.db.Where(&serverConfig{DiscordId: guildId}).Find(&sc)
	if (sc == serverConfig{}) {
		return defaultServerConfig
	}
	return sc
}

func (bot *amputatorBot) setServerConfig(s *discordgo.Session, m *discordgo.Message) error {
	sc := bot.getServerConfig(m.GuildID)
	if sc == defaultServerConfig {
		return fmt.Errorf("unable to look up server config for guild: %v", m.GuildID)
	}

	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return fmt.Errorf("unable to look up guild by id: %v", m.GuildID)
	}

	command := strings.Split(m.Content, " ")
	setting := command[2]
	value := command[3]

	errorEmbed := &discordgo.MessageEmbed{
		Title:       "Unable to set " + value,
		Description: "See https://github.com/tyzbit/go-discord-amputator for usage",
	}

	// TODO: make this not use raw fields
	log.Debug(fmt.Sprintf("%#v", sc))
	switch setting {
	case "switch":
		bot.db.Model(&serverConfig{}).Where(&serverConfig{DiscordId: guild.ID}).Update("amputation_enabled", value == "on")
	case "replyto":
		bot.db.Model(&serverConfig{}).Where(&serverConfig{DiscordId: guild.ID}).Update("reply_to_original_message", value == "on")
	case "embed":
		bot.db.Model(&serverConfig{}).Where(&serverConfig{DiscordId: guild.ID}).Update("use_embed", value == "on")
	case "guess":
		bot.db.Model(&serverConfig{}).Where(&serverConfig{DiscordId: guild.ID}).Update("guess_and_check", value == "on")
	case "maxdepth":
		maxDepth, err := strconv.Atoi(value)
		if err != nil {
			bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m, errorEmbed)
		}
		sc.MaxDepth = maxDepth
		bot.db.Model(&serverConfig{}).Where(&serverConfig{DiscordId: guild.ID}).Updates(&sc)
	default:
		bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m, errorEmbed)
		return nil
	}

	bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m, &discordgo.MessageEmbed{
		Title:       "Setting Updated",
		Description: setting + " set to " + value,
	})

	return nil
}

// updateServersWatched updates the servers watched value
// in both the local bot stats and in the database. It is allowed to fail.
func (bot *amputatorBot) updateServersWatched(s *discordgo.Session) error {
	stats := bot.getGlobalStats()

	usd := &discordgo.UpdateStatusData{Status: "online"}
	usd.Activities = make([]*discordgo.Activity, 1)
	usd.Activities[0] = &discordgo.Activity{
		Name: fmt.Sprintf("%v servers", stats.ServersWatched),
		Type: discordgo.ActivityTypeWatching,
		URL:  "https://github.com/tyzbit/go-discord-amputator",
	}

	err := s.UpdateStatusComplex(*usd)
	if err != nil {
		return fmt.Errorf("unable to update discord status: %w", err)
	}

	return nil
}
