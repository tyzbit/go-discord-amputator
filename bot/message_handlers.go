package bot

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
func (bot *AmputatorBot) handleMessageWithStats(s *discordgo.Session, m *discordgo.MessageCreate) error {
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
		// We can be sure now the request was a direct message.
		// Deny by default.
		administrator := false

	out:
		for _, id := range bot.Config.AdminIds {
			if m.Author.ID == id {
				administrator = true

				// This prevents us from checking all IDs now that
				// we found a match but is a fairly ineffectual
				// optimization since config.AdminIds will probably
				// only have dozens of IDs at most.
				break out
			}
		}

		if !administrator {
			return fmt.Errorf("did not respond to %v(%v), command %v because user is not an administrator",
				m.Author.Username, m.Author.ID, statsCommand)
		}
		stats = bot.getGlobalStats()
		logMessage = "sending global " + statsCommand + " response to " + m.Author.Username + "(" + m.Author.ID + ")"
	}

	// write a new statsMessageEvent to the DB
	bot.createMessageEvent(statsCommand, m.Message)

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
func (bot *AmputatorBot) handleMessageWithAmpUrls(s *discordgo.Session, m *discordgo.MessageCreate) error {
	ServerConfig := bot.getServerConfig(m.GuildID)
	if !ServerConfig.AmputationEnabled {
		log.Info("URLs were not amputated because automatic amputation is not enabled")
		return nil
	}

	xurlsStrict := xurls.Strict
	urls := xurlsStrict.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		return fmt.Errorf("found 0 URLs in message that matched amp regex: %v", ampRegex)
	}

	log.Debug("URLs parsed from message: ", strings.Join(urls, ", "))

	// This UUID will be used to tie together the AmputationEvent,
	// the amputationRequestUrls and the amputationResponseUrls.
	ampEventUUID := uuid.New().String()

	var amputations []Amputation
	for _, url := range urls {
		domainName, err := getDomainName(url)
		if err != nil {
			log.Error("unable to get domain name for url: ", url)
		}

		cachedAmputations := []Amputation{}
		bot.DB.Model(&Amputation{}).Where(&Amputation{RequestURL: url, Cached: false}).Find(&cachedAmputations)
		var responseUrl, responseDomainName string

		for _, cachedAmputation := range cachedAmputations {
			if cachedAmputation.ResponseURL != "" && cachedAmputation.ResponseDomainName != "" {
				responseUrl = cachedAmputation.ResponseURL
				responseDomainName = cachedAmputation.ResponseDomainName
			}
		}

		if responseUrl != "" && responseDomainName != "" {
			log.Debug("url was already cached: ", url)
			// We have already amputated this URL, so save the response
			amputations = append(amputations, Amputation{
				UUID:                uuid.New().String(),
				AmputationEventUUID: ampEventUUID,
				RequestURL:          url,
				RequestDomainName:   domainName,
				ResponseURL:         responseUrl,
				ResponseDomainName:  responseDomainName,
				Cached:              true,
			})
			continue
		}

		// We have not already amputated this URL, so build an object
		// for doing so.
		log.Debug("url was not cached: ", url)
		amputations = append(amputations, Amputation{
			UUID:                uuid.New().String(),
			AmputationEventUUID: ampEventUUID,
			RequestURL:          url,
			RequestDomainName:   domainName,
			Cached:              false,
		})
	}

	var amputatedLinks []string
	for i, amputation := range amputations {
		if amputation.ResponseURL == "" {
			log.Debug("need to call amputator api for ", amputation.RequestURL)
			amputatedUrls, err := goamputate.Amputate([]string{amputation.RequestURL}, map[string]string{
				"gac": fmt.Sprintf("%v", ServerConfig.GuessAndCheck),
				"md":  fmt.Sprintf("%v", ServerConfig.MaxDepth),
			})
			if err != nil {
				log.Error("error calling amputator api: ", err)
				continue
			}
			if !(len(amputatedUrls) == 1) {
				log.Errorf("received %v urls from goamputate, expected 1", len(amputatedUrls))
				continue
			}
			domainName, err := getDomainName(amputatedUrls[0])
			if err != nil {
				log.Errorf("unable to get domain name for url: %v", amputation.ResponseURL)
			}
			amputations[i].ResponseURL = amputatedUrls[0]
			amputations[i].ResponseDomainName = domainName
			amputatedLinks = append(amputatedLinks, amputatedUrls[0])
			continue
		}
		// We have a response URL, so add that to the links to be used
		// in the message.
		amputatedLinks = append(amputatedLinks, amputation.ResponseURL)
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
	bot.sendMessage(s, ServerConfig.UseEmbed, ServerConfig.ReplyToOriginalMessage, m.Message, embed)

	// Create a call to Amputator API event
	tx := bot.DB.Create(&AmputationEvent{
		UUID:           ampEventUUID,
		AuthorId:       m.Author.ID,
		AuthorUsername: m.Author.Username,
		ChannelId:      m.ChannelID,
		MessageId:      m.ID,
		ServerId:       guild.ID,
		Amputations:    amputations,
	})

	if tx.RowsAffected != 1 {
		return fmt.Errorf("unexpected number of rows affected inserting amputation event: %v", tx.RowsAffected)
	}

	return nil
}
