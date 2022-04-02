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
	logMessage := ""
	if !directMessage {
		stats = bot.getServerStats(m.GuildID)
		guild, err := s.Guild(m.GuildID)
		if err != nil {
			return fmt.Errorf("unable to look up guild by id: %v", m.GuildID+", "+fmt.Sprintf("%v", err))
		}
		logMessage = "sending " + statsCommand + " response to " + m.Author.Username + "(" + m.Author.ID + ") in " +
			guild.Name + "(" + guild.ID + ")"
	} else {
		stats = bot.getGlobalStats()
		logMessage = "sending global " + statsCommand + " response to " + m.Author.Username + "(" + m.Author.ID + ")"
	}

	// write a new statsMessageEvent to the DB
	bot.createMessageEvent(statsCommand, m.Message)

	// We can be sure now the request was a direct message.
	// Deny by default.
	administrator := false

out:
	for _, id := range config.adminIds {
		if m.Author.ID == id {
			administrator = true

			// This prevents us from checking all IDs now that
			// we found a match but is a fairly ineffectual
			// optimization since config.adminIds will probably
			// only have dozens of IDs at most.
			break out
		}
	}

	if !administrator {
		return fmt.Errorf("did not respond to %v(%v), command %v because user is not an administrator",
			m.Author.Username, m.Author.ID, statsCommand)
	}

	embed := &discordgo.MessageEmbed{
		Title:  "Amputation Stats",
		Fields: structToPrettyDiscordFields(stats),
	}

	// Respond to statsCommand command with the formatted stats embed
	log.Info(logMessage)
	bot.sendMessage(s, true, false, m.Message, embed)

	return nil
}

// handleMessageWithAmpUrls takes a Discord session and a message string and
// calls go-amputator with a []string of URLs parsed from the message.
// It then sends an embed with the resulting amputated URLs.
func (bot *amputatorBot) handleMessageWithAmpUrls(s *discordgo.Session, m *discordgo.MessageCreate) error {
	serverConfig := bot.getServerConfig(m.GuildID)
	if !serverConfig.AmputationEnabled {
		log.Info("URLs were not amputated because automatic amputation is not enabled")
		return nil
	}

	xurlsStrict := xurls.Strict
	urls := xurlsStrict.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		return fmt.Errorf("found 0 URLs in message that matched amp regex: %v", ampRegex)
	}

	log.Debug("URLs parsed from message: ", strings.Join(urls, ", "))

	// This UUID will be used to tie together the amputationEvent,
	// the amputationRequestUrls and the amputationResponseUrls.
	ampEventUUID := uuid.New().String()

	// Create the list of URLs requested for this call to the Amputator API
	var ampRequestUrls []urlInfo
	for _, url := range urls {
		domainName, err := getDomainName(url)
		if err != nil {
			log.Warn("unable to get domain name for url: ", url)
			domainName = ""
		}

		ampRequestUrls = append(ampRequestUrls, urlInfo{
			UUID:                uuid.New().String(),
			AmputationEventUUID: ampEventUUID,
			Type:                "request",
			URL:                 url,
			DomainName:          domainName,
		})
	}

	// initialize and call the amputator API
	amputator := goamputate.AmputatorBot{}
	amputatedLinks, err := amputator.Amputate(urls, map[string]string{
		"gac": fmt.Sprintf("%v", serverConfig.GuessAndCheck),
		"md":  fmt.Sprintf("%v", serverConfig.MaxDepth),
	})

	if err != nil || len(amputatedLinks) == 0 {
		bot.sendMessage(s, serverConfig.UseEmbed, serverConfig.ReplyToOriginalMessage,
			m.Message, &discordgo.MessageEmbed{
				Title:       "Problem Amputating",
				Description: "Sorry, I couldn't amputate that link.",
			})
		return err
	}

	// Create the list of URLs we got back for this call to the Amputator API
	var ampResponseUrls []urlInfo
	for _, url := range amputatedLinks {
		domainName, err := getDomainName(url)
		if err != nil {
			log.Warn("unable to get domain name for url: ", url)
			domainName = ""
		}

		ampResponseUrls = append(ampResponseUrls, urlInfo{
			UUID:                uuid.New().String(),
			Type:                "response",
			DomainName:          domainName,
			AmputationEventUUID: ampEventUUID,
			URL:                 url,
		})
	}

	// Do a lookup for the full guild object
	guild, gErr := s.Guild(m.GuildID)
	if gErr != nil {
		return fmt.Errorf("unable to look up guild by id: %v", m.GuildID)
	}

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
	bot.sendMessage(s, serverConfig.UseEmbed, serverConfig.ReplyToOriginalMessage, m.Message, embed)

	// Create a call to Amputator API event
	tx := bot.db.Create(&amputationEvent{
		UUID:           ampEventUUID,
		AuthorId:       m.Author.ID,
		AuthorUsername: m.Author.Username,
		ChannelId:      m.ChannelID,
		MessageId:      m.ID,
		ServerId:       guild.ID,
		RequestURLs:    ampRequestUrls,
		ResponseURLs:   ampResponseUrls,
	})

	if tx.RowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected inserting amputation event: %v", tx.RowsAffected)
	}

	return nil
}
