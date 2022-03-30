package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// Watches the infoUpdates channel (botInfo type) and updates bot info
func (bot *amputatorBot) statsHandler() {
	for stats := range bot.infoUpdates {
		bot.info = stats
	}
}

// updateMessagesSeen updates the messages seen value in both the local bot
// stats and in the database
func (bot *amputatorBot) updateMessagesSeen(i int) {
	localStats := bot.info
	localStats.MessagesSeen = i
	bot.infoUpdates <- localStats

	field := getBotInfoTagValue("db", "MessagesSeen")
	if field == "" {
		log.Error("db tag was blank for MessagesSeen")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// updateMessagesActedOn updates the messages acted on value in both the
// local bot stats and in the database
func (bot *amputatorBot) updateMessagesActedOn(i int) {
	localStats := bot.info
	localStats.MessagesActedOn = i
	bot.infoUpdates <- localStats

	field := getBotInfoTagValue("db", "MessagesActedOn")
	if field == "" {
		log.Error("db tag was blank for MessagesActedOn")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// updateMessagesSent updates the messages sent value in both the
// local bot stats and in the database
func (bot *amputatorBot) updateMessagesSent(i int) {
	localStats := bot.info
	localStats.MessagesSent = i
	bot.infoUpdates <- localStats

	field := getBotInfoTagValue("db", "MessagesSent")
	if field == "" {
		log.Error("db tag was blank for MessagesSent")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// CallsToAmputatorAPI updates the calls to Amputator API value
// in both the local bot stats and in the database
func (bot *amputatorBot) updateCallsToAmputatorApi(i int) {
	localStats := bot.info
	localStats.CallsToAmputatorAPI = i
	bot.infoUpdates <- localStats

	field := getBotInfoTagValue("db", "CallsToAmputatorAPI")
	if field == "" {
		log.Error("db tag was blank for CallsToAmputatorAPI")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}

// URLsAmputated updates the URLs amputated value
// in both the local bot stats and in the database
func (bot *amputatorBot) updateUrlsAmputated(i int) {
	localStats := bot.info
	localStats.URLsAmputated = i
	bot.infoUpdates <- localStats

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
	localStats := bot.info
	localStats.ServersWatched = i
	bot.infoUpdates <- localStats
	if field == "" {
		log.Error("db tag was blank for ServersWatched")
		return
	}
	bot.dbUpdates <- field + " = " + fmt.Sprintf("%v", i)
}
