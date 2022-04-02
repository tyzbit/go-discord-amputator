package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/mvdan/xurls"
	log "github.com/sirupsen/logrus"
	goamputate "github.com/tyzbit/go-amputate"
)

// handleMessageWithStats takes a discord session and a user ID and sends a
// message to the user with stats about the bot.
func (bot *amputatorBot) handleMessageWithStats(s *discordgo.Session, m *discordgo.MessageCreate) error {
	directMessage := (m.GuildID == "")

	var stats botStats
	statsLogMessage := ""
	if !directMessage {
		stats = bot.getServerStats(m.GuildID)
		guild, err := s.Guild(m.GuildID)
		if err != nil {
			return fmt.Errorf("unable to look up guild by id: %v", m.GuildID+", "+fmt.Sprintf("%v", err))
		}
		statsLogMessage = "sending " + statsCommand + " response to " + m.Author.Username + "(" + m.Author.ID + ") in " +
			guild.Name + "(" + guild.ID + ")"
	} else {
		stats = bot.getGlobalStats()
		statsLogMessage = "sending global " + statsCommand + " response to " + m.Author.Username + "(" + m.Author.ID + ")"
	}

	// write a new statsMessageEvent to the DB
	bot.createMessageEvent(statsCommand, m.Message)

	// We can be sure now the request was a direct message
	administrator := false

out:
	for _, id := range config.adminIds {
		if m.Author.ID == id {
			administrator = true
			break out
		}
	}

	if !administrator {
		log.Info()
		return fmt.Errorf("did not respond to %v(%v) %v command because user is not an administrator",
			m.Author.Username, m.Author.ID, statsCommand)
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Amputation Stats",
		Fields: structToDiscordFields(stats),
	}

	// Respond to statsCommand command with the formatted stats embed
	log.Info(statsLogMessage)
	bot.sendMessage(s, true, false, m.Message, embed)

	return nil
}

// handleMessageWithAmpUrls takes a Discord session and a message string and
// calls go-amputator with a []string of URLs parsed from the message.
// It then sends an embed with the resulting amputated URLs.
func (bot *amputatorBot) handleMessageWithAmpUrls(s *discordgo.Session, m *discordgo.MessageCreate) error {
	sc := bot.getServerConfig(m.GuildID)
	if !sc.AmputationEnabled {
		log.Info("URLs were not amputated because automatic amputation is not enabled")
		return nil
	}

	xurlsStrict := xurls.Strict
	urls := xurlsStrict.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		log.Debug("found 0 URLs in message that matched amp regex: ", ampRegex)
		return nil
	}

	log.Debug("URLs parsed from message: ", strings.Join(urls, ", "))

	// This UUID will be used to tie together the amputationEvent,
	// the amputationRequestUrl and the amputationResponseUrl.
	ampEventUUID := uuid.New().String()

	// Create the list of URLs requested for this call to the Amputator API
	var ampRequestUrls []urlInfo
	for _, url := range urls {
		ampRequestUrls = append(ampRequestUrls, urlInfo{
			UUID:                uuid.New().String(),
			AmputationEventUUID: ampEventUUID,
			Type:                "request",
			URL:                 url,
			DomainName:          getDomainName(url),
		})
	}

	genericLinkAmputationFailureMessage := &discordgo.MessageEmbed{
		Title:       "Problem Amputating",
		Description: "Sorry, I couldn't amputate that link.",
	}

	amputator := goamputate.AmputatorBot{}
	amputatedLinks, err := amputator.Amputate(urls, map[string]string{
		"gac": fmt.Sprintf("%v", sc.GuessAndCheck),
		"md":  fmt.Sprintf("%v", sc.MaxDepth),
	})

	if err != nil || len(amputatedLinks) == 0 {
		bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage,
			m.Message, genericLinkAmputationFailureMessage)
	}

	// Create the list of URLs we got back for this call to the Amputator API
	var ampResponseUrls []urlInfo
	for _, url := range amputatedLinks {
		ampResponseUrls = append(ampResponseUrls, urlInfo{
			UUID:                uuid.New().String(),
			Type:                "response",
			DomainName:          getDomainName(url),
			AmputationEventUUID: ampEventUUID,
			URL:                 url,
		})
	}

	// TODO: There has to be a better way, but the guild name is blank in the s and g objects
	guild, gErr := s.Guild(m.GuildID)
	if gErr != nil {
		log.Error("unable to look up guild by id: ", m.GuildID)
	}

	// Create a call to Amputator API event
	bot.db.Create(&amputationEvent{
		UUID:           ampEventUUID,
		AuthorId:       m.Author.ID,
		AuthorUsername: m.Author.Username,
		ChannelId:      m.ChannelID,
		MessageId:      m.ID,
		ServerId:       guild.ID,
		RequestURLs:    ampRequestUrls,
		ResponseURLs:   ampResponseUrls,
	})

	plural := ""
	if len(amputatedLinks) > 1 {
		plural = "s"
	}
	title := fmt.Sprintf("Amputated Link%v", plural)

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: strings.Join(amputatedLinks, "\n"),
	}

	log.Debug("sending amputate message response in ",
		guild.Name, "(", m.GuildID, "), calling user: ",
		m.Author.Username, "(", m.Author.ID, ")")
	bot.sendMessage(s, sc.UseEmbed, sc.ReplyToOriginalMessage, m.Message, embed)

	return nil
}
