package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/mvdan/xurls"
	"github.com/sirupsen/logrus"
	goamputate "github.com/tyzbit/go-amputate"
)

// handleMessageWithStats takes a discord session and a user ID and sends a
// message to the user with stats about the bot.
func (bot amputatorBot) handleMessageWithStats(s *discordgo.Session, m *discordgo.MessageCreate) {
	administrator := false
	for _, id := range strings.Split(env[adminIds], ",") {
		if m.Author.ID == id {
			administrator = true
		}
	}

	if administrator {
		formattedStats := ""
		keys := make([]string, 0, len(bot.stats))
		for k := range bot.stats {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			formattedStats = fmt.Sprintf("%v\n%v: %v", formattedStats, k, bot.stats[k])
		}
		bot.stats["messagesSent"]++
		bot.updateMessagesSent(bot.stats["messagesSent"])
		embed := &discordgo.MessageEmbed{
			Title:       "Amputation Stats",
			Description: formattedStats,
		}

		logrus.Debug("sending !stats response to ", m.Author.Username, "(", m.Author.ID, ")")
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			logrus.Error("unable to send embed: ", err)
		}
	} else {
		logrus.Debug("did not respond to ", m.Author.Username, " id ", m.Author.ID, " because user is not an administrator")
	}
}

// handleMessageWithAmpUrls takes a Discord session and a message string and
// calls go-amputator with a []string of URLs parsed from the message.
// It then sends an embed with the resulting amputated URLs.
func (bot amputatorBot) handleMessageWithAmpUrls(s *discordgo.Session, m *discordgo.MessageCreate) {
	xurlsRelaxed := xurls.Strict
	urls := xurlsRelaxed.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		logrus.Debug("found 0 URLs in message that matched amp regex: ", ampRegex)
		return
	}

	logrus.Debug("URLs parsed from message: ", strings.Join(urls, ", "))

	var amputator goamputate.AmputatorBot
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
	bot.stats["callsToAmputatorApi"]++
	bot.updateCallsToAmputatorApi(bot.stats["callsToAmputatorApi"])
	amputatedLinks, err := amputator.Amputate(urls, options)
	if err != nil {
		logrus.Error("error calling Amputator API: ", err)
		bot.stats["messagesSent"]++
		bot.updateMessagesSent(bot.stats["messagesSent"])
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, genericLinkAmputationFailureMessage)
		if err != nil {
			logrus.Error("unable to send embed: ", err)
		}
		return
	}

	if len(amputatedLinks) == 0 {
		logrus.Warn("amputator bot returned no Amputated URLs from: ", strings.Join(urls, ", "))
		bot.stats["messagesSent"]++
		bot.updateMessagesSent(bot.stats["messagesSent"])
		_, err := s.ChannelMessageSendEmbed(m.ChannelID, genericLinkAmputationFailureMessage)
		if err != nil {
			logrus.Error("unable to send embed: ", err)
		}
		return
	}
	bot.stats["urlsAmputated"] = bot.stats["urlsAmputated"] + len(amputatedLinks)
	bot.updateUrlsAmputated(bot.stats["urlsAmputated"])

	plural := ""
	if len(amputatedLinks) > 1 {
		plural = "s"
	}

	title := fmt.Sprintf("Amputated Link%v", plural)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: strings.Join(amputatedLinks, "\n"),
	}
	bot.stats["messagesSent"]++
	bot.updateMessagesSent(bot.stats["messagesSent"])
	guild, err := s.Guild(m.GuildID)
	guildName := "unknown"
	if err != nil {
		logrus.Warn("couldn't get guild for ID ", m.GuildID)
	}
	if guild.Name != "" {
		guildName = guild.Name
	}
	logrus.Debug("sending amputate message response in ", guildName, "(", m.GuildID, "), calling user: ", m.Author.Username, "(", m.Author.ID, ")")
	_, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
	if err != nil {
		logrus.Error("unable to send embed: ", err)
	}
}
