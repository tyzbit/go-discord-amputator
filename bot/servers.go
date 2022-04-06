package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ServerRegistration struct {
	DiscordId string `gorm:"primaryKey"`
	Name      string
	UpdatedAt time.Time
	Config    ServerConfig `gorm:"foreignKey:DiscordId"`
}

type ServerConfig struct {
	DiscordId              string `gorm:"primaryKey" pretty:"Server ID"`
	Name                   string `pretty:"Server Name"`
	AmputationEnabled      bool   `pretty:"Amputation Enabled"`
	ReplyToOriginalMessage bool   `pretty:"Reply to original message"`
	UseEmbed               bool   `pretty:"Use embed to reply"`
	GuessAndCheck          bool   `pretty:"Guess at AMP URLs if they are difficult"`
	MaxDepth               int    `pretty:"How many links deep to go to try to find the non-AMP link"`
}

var (
	defaultServerConfig ServerConfig = ServerConfig{
		DiscordId:              "0",
		Name:                   "default",
		AmputationEnabled:      true,
		ReplyToOriginalMessage: false,
		UseEmbed:               true,
		GuessAndCheck:          true,
		MaxDepth:               3,
	}

	amputatorRepoUrl string = "https://github.com/tyzbit/go-discord-amputator"
)

// registerOrUpdateGuild checks if a guild is already registered in the database. If not,
// it creates it with sensibile defaults.
func (bot *AmputatorBot) registerOrUpdateGuild(s *discordgo.Session, g *discordgo.Guild) error {
	var registration ServerRegistration
	bot.DB.Find(&registration, g.ID)

	// Do a lookup for the full guild object
	guild, err := s.Guild(g.ID)
	if err != nil {
		return fmt.Errorf("unable to look up guild by id: %v", g.ID)
	}

	// The server registration does not exist, so we will create with defaults
	if (registration == ServerRegistration{}) {
		log.Info("creating registration for new server: ", guild.Name, "(", g.ID, ")")
		tx := bot.DB.Create(&ServerRegistration{
			DiscordId: g.ID,
			Name:      guild.Name,
			UpdatedAt: time.Now(),
			Config:    defaultServerConfig,
		})

		// We only expect one server to be updated at a time. Otherwise, return an error.
		if tx.RowsAffected != 1 {
			return fmt.Errorf("did not expect %v rows to be affected updating "+
				"server registration for server: %v(%v)", fmt.Sprintf("%v", tx.RowsAffected), guild.Name, g.ID)
		}
	}

	err = bot.updateServersWatched(s)
	if err != nil {
		return fmt.Errorf("unable to update servers watched: %v", err)
	}

	return nil
}

// getServerConfig takes a guild ID and returns a ServerConfig object for that server.
// If the config isn't found, it returns a default config.
func (bot *AmputatorBot) getServerConfig(guildId string) ServerConfig {
	sc := ServerConfig{}
	bot.DB.Where(&ServerConfig{DiscordId: guildId}).Find(&sc)
	if (sc == ServerConfig{}) {
		return defaultServerConfig
	}
	return sc
}

// setServerConfig sets a single config setting for the calling server. Syntax:
// (commandPrefix) config [setting] [value]
func (bot *AmputatorBot) setServerConfig(s *discordgo.Session, m *discordgo.Message) error {
	sc := bot.getServerConfig(m.GuildID)
	if sc == defaultServerConfig {
		return fmt.Errorf("unable to look up server config for guild: %v", m.GuildID)
	}

	guild, err := s.Guild(m.GuildID)
	if err != nil {
		return fmt.Errorf("unable to look up guild by id: %v", m.GuildID)
	}

	command := strings.Split(m.Content, " ")
	var setting, value string
	if len(command) == 4 {
		setting = command[2]
		value = command[3]
	} else {
		setting = "get"
	}

	errorEmbed := &discordgo.MessageEmbed{
		Title:       "Unable to set " + value,
		Description: "See " + amputatorRepoUrl + " for usage",
	}

	tx := &gorm.DB{}
	switch setting {
	// "get" is the only command that does not alter the database.
	case "get":
		bot.sendMessage(s, true, false, m, &discordgo.MessageEmbed{
			Title:  "Amputator Config",
			Fields: structToPrettyDiscordFields(sc),
		})
		return nil
	case "switch":
		tx = bot.DB.Model(&ServerConfig{}).Where(&ServerConfig{DiscordId: guild.ID}).Update("amputation_enabled", value == "on")
	case "replyto":
		tx = bot.DB.Model(&ServerConfig{}).Where(&ServerConfig{DiscordId: guild.ID}).Update("reply_to_original_message", value == "on")
	case "embed":
		tx = bot.DB.Model(&ServerConfig{}).Where(&ServerConfig{DiscordId: guild.ID}).Update("use_embed", value == "on")
	case "guess":
		tx = bot.DB.Model(&ServerConfig{}).Where(&ServerConfig{DiscordId: guild.ID}).Update("guess_and_check", value == "on")
	case "maxdepth":
		maxDepth, err := strconv.Atoi(value)
		if err != nil {
			bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m, errorEmbed)
			return fmt.Errorf("unable to convert max depth from string to integer")
		}
		tx = bot.DB.Model(&ServerConfig{}).Where(&ServerConfig{DiscordId: guild.ID}).Update("max_depth", maxDepth)
	default:
		bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m, errorEmbed)
		return nil
	}

	// We only expect one server to be updated at a time. Otherwise, return an error.
	if tx.RowsAffected != 1 {
		return fmt.Errorf("did not expect %v rows to be affected updating "+
			"server config for server: %v(%v)", fmt.Sprintf("%v", tx.RowsAffected), guild.Name, guild.ID)
	}

	bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m, &discordgo.MessageEmbed{
		Title:       "Setting Updated",
		Description: setting + " set to " + value,
	})

	return nil
}

// updateServersWatched updates the servers watched value
// in both the local bot stats and in the database. It is allowed to fail.
func (bot *AmputatorBot) updateServersWatched(s *discordgo.Session) error {
	var serversWatched int64
	bot.DB.Model(&ServerRegistration{}).Where(&ServerRegistration{}).Count(&serversWatched)

	updateStatusData := &discordgo.UpdateStatusData{Status: "online"}
	updateStatusData.Activities = make([]*discordgo.Activity, 1)
	updateStatusData.Activities[0] = &discordgo.Activity{
		Name: fmt.Sprintf("%v servers", serversWatched),
		Type: discordgo.ActivityTypeWatching,
		URL:  amputatorRepoUrl,
	}

	if !bot.StartingUp {
		log.Debug("updating discord bot status")
		err := s.UpdateStatusComplex(*updateStatusData)
		if err != nil {
			return fmt.Errorf("unable to update discord bot status: %w", err)
		}
	}

	return nil
}
