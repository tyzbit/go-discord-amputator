package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// updateMessagesSeen updates the messages seen value in both the local bot
// stats and in the database
func (bot *amputatorBot) setMessagesSeen(i int) {
	bot.infoUpdates.Lock()
	bot.info.MessagesSeen = i
	bot.infoUpdates.Unlock()

	field := getBotInfoTagValue("db", "MessagesSeen")
	if field == "" {
		log.Error("db tag was blank for MessagesSeen")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// updateMessagesActedOn updates the messages acted on value in both the
// local bot stats and in the database
func (bot *amputatorBot) setMessagesActedOn(i int) {
	bot.infoUpdates.Lock()
	bot.info.MessagesActedOn = i
	bot.infoUpdates.Unlock()

	field := getBotInfoTagValue("db", "MessagesActedOn")
	if field == "" {
		log.Error("db tag was blank for MessagesActedOn")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// updateMessagesSent updates the messages sent value in both the
// local bot stats and in the database
func (bot *amputatorBot) setMessagesSent(i int) {
	bot.infoUpdates.Lock()
	bot.info.MessagesSent = i
	bot.infoUpdates.Unlock()

	field := getBotInfoTagValue("db", "MessagesSent")
	if field == "" {
		log.Error("db tag was blank for MessagesSent")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// CallsToAmputatorAPI updates the calls to Amputator API value
// in both the local bot stats and in the database
func (bot *amputatorBot) setCallsToAmputatorApi(i int) {
	bot.infoUpdates.Lock()
	bot.info.CallsToAmputatorAPI = i
	bot.infoUpdates.Unlock()

	field := getBotInfoTagValue("db", "CallsToAmputatorAPI")
	if field == "" {
		log.Error("db tag was blank for CallsToAmputatorAPI")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// URLsAmputated updates the URLs amputated value
// in both the local bot stats and in the database
func (bot *amputatorBot) setUrlsAmputated(i int) {
	bot.infoUpdates.Lock()
	bot.info.URLsAmputated = i
	bot.infoUpdates.Unlock()

	field := getBotInfoTagValue("db", "URLsAmputated")
	if field == "" {
		log.Error("db tag was blank for URLsAmputated")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// updateServersWatched updates the servers watched value
// in both the local bot stats and in the database
func (bot *amputatorBot) updateServersWatched(s *discordgo.Session, i int) {
	log.Info("watching ", i, " servers")
	usd := &discordgo.UpdateStatusData{Status: "online"}
	usd.Activities = make([]*discordgo.Activity, 1)
	usd.Activities[0] = &discordgo.Activity{
		Name: fmt.Sprintf("%v servers", i),
		Type: discordgo.ActivityTypeWatching,
		URL:  "https://github.com/tyzbit/go-discord-amputator",
	}

	err := s.UpdateStatusComplex(*usd)
	if err != nil {
		log.Error("failed to set status: ", err)
	}

	field := getBotInfoTagValue("db", "ServersWatched")
	bot.infoUpdates.Lock()
	bot.info.ServersWatched = i
	bot.infoUpdates.Unlock()
	if field == "" {
		log.Error("db tag was blank for ServersWatched")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}
