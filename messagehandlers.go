package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mvdan/xurls"
	log "github.com/sirupsen/logrus"
	goamputate "github.com/tyzbit/go-amputate"
)

// handleMessageWithStats takes a discord session and a user ID and sends a
// message to the user with stats about the bot.
func (bot *amputatorBot) handleMessageWithStats(s *discordgo.Session, m *discordgo.MessageCreate) {
	administrator := false
	for _, id := range config.adminIds {
		if m.Author.ID == id {
			administrator = true
		}
	}

	if administrator {
		formattedStats := getTagValueByTag("pretty", "serversWatched") + ": " + fmt.Sprintf("%v", bot.currentStats.serversWatched) + "\n" +
			getTagValueByTag("pretty", "messagesSeen") + ": " + fmt.Sprintf("%v", bot.currentStats.messagesSeen) + "\n" +
			getTagValueByTag("pretty", "messagesSent") + ": " + fmt.Sprintf("%v", bot.currentStats.messagesSent) + "\n" +
			getTagValueByTag("pretty", "messagesActedOn") + ": " + fmt.Sprintf("%v", bot.currentStats.messagesActedOn) + "\n" +
			getTagValueByTag("pretty", "callsToAmputatorApi") + ": " + fmt.Sprintf("%v", bot.currentStats.callsToAmputatorApi) + "\n" +
			getTagValueByTag("pretty", "urlsAmputated") + ": " + fmt.Sprintf("%v", bot.currentStats.urlsAmputated) + "\n"

		bot.updateMessagesSent(bot.currentStats.messagesSent + 1)
		embed := &discordgo.MessageEmbed{
			Title:       "Amputation Stats",
			Description: formattedStats,
		}

		log.Debug("sending !stats response to ", m.Author.Username, "(", m.Author.ID, ")")
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			log.Error("unable to send embed: ", err)
		}
	} else {
		log.Debug("did not respond to ", m.Author.Username, " id ", m.Author.ID, " because user is not an administrator")
	}
}

// handleMessageWithAmpUrls takes a Discord session and a message string and
// calls go-amputator with a []string of URLs parsed from the message.
// It then sends an embed with the resulting amputated URLs.
func (bot *amputatorBot) handleMessageWithAmpUrls(s *discordgo.Session, m *discordgo.MessageCreate) {
	xurlsRelaxed := xurls.Strict
	urls := xurlsRelaxed.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		log.Debug("found 0 URLs in message that matched amp regex: ", ampRegex)
		return
	}

	log.Debug("URLs parsed from message: ", strings.Join(urls, ", "))

	var amputator goamputate.AmputatorBot
	options := map[string]string{}

	// Read environment options and set parameters appropriately.
	// These are string and not bool and int because bool and int zero values
	// are false and 0, which are valid selections so we can't be positive
	// those weren't actively selected.
	if config.guessAndCheck != "" {
		options["gac"] = config.guessAndCheck
	}
	if config.maxDepth != "" {
		options["md"] = config.maxDepth
	}

	genericLinkAmputationFailureMessage := &discordgo.MessageEmbed{
		Title:       "Problem Amputating",
		Description: "Sorry, I couldn't amputate that link.",
	}
	bot.updateCallsToAmputatorApi(bot.currentStats.callsToAmputatorApi + 1)
	amputatedLinks, err := amputator.Amputate(urls, options)
	if err != nil {
		log.Error("error calling Amputator API: ", err)
		bot.updateMessagesSent(bot.currentStats.messagesSent + 1)
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, genericLinkAmputationFailureMessage)
		if err != nil {
			log.Error("unable to send embed: ", err)
		}
		return
	}

	if len(amputatedLinks) == 0 {
		log.Warn("amputator bot returned no Amputated URLs from: ", strings.Join(urls, ", "))
		bot.updateMessagesSent(bot.currentStats.messagesSent + 1)
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, genericLinkAmputationFailureMessage)
		if err != nil {
			log.Error("unable to send embed: ", err)
		}
		return
	}
	bot.updateUrlsAmputated(bot.currentStats.urlsAmputated + 1)

	plural := ""
	if len(amputatedLinks) > 1 {
		plural = "s"
	}

	title := fmt.Sprintf("Amputated Link%v", plural)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: strings.Join(amputatedLinks, "\n"),
	}

	guild, err := s.Guild(m.GuildID)
	guildName := "unknown"
	if err != nil {
		log.Warn("couldn't get guild for ID ", m.GuildID)
	}
	if guild.Name != "" {
		guildName = guild.Name
	}
	log.Debug("sending amputate message response in ", guildName, "(", m.GuildID, "), calling user: ", m.Author.Username, "(", m.Author.ID, ")")
	_, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		log.Error("unable to send embed: ", err)
	}
	bot.updateMessagesSent(bot.currentStats.messagesSent + 1)
}
