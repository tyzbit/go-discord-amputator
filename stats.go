package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// updateMessagesSeen updates the messages seen value in both the local bot
// stats and in the database
func (bot *amputatorBot) updateMessagesSeen(i int) {
	field := getTagValueByTag("sql", "messagesSeen")
	localStats := bot.currentStats
	localStats.messagesSeen = i
	bot.statsChannel <- localStats
	if field == "" {
		log.Error("sql tag was blank for messagesSeen")
		return
	}
	bot.dbChannel <- field + " = " + fmt.Sprintf("%v", i)
}

// updateMessagesActedOn updates the messages acted on value in both the
// local bot stats and in the database
func (bot *amputatorBot) updateMessagesActedOn(i int) {
	field := getTagValueByTag("sql", "messagesActedOn")
	localStats := bot.currentStats
	localStats.messagesActedOn = i
	bot.statsChannel <- localStats
	if field == "" {
		log.Error("sql tag was blank for messagesActedOn")
		return
	}
	bot.dbChannel <- field + " = " + fmt.Sprintf("%v", i)
}

// updateMessagesSent updates the messages sent value in both the
// local bot stats and in the database
func (bot *amputatorBot) updateMessagesSent(i int) {
	field := getTagValueByTag("sql", "messagesSent")
	localStats := bot.currentStats
	localStats.messagesSent = i
	bot.statsChannel <- localStats
	if field == "" {
		log.Error("sql tag was blank for messagesSent")
		return
	}
	bot.dbChannel <- field + " = " + fmt.Sprintf("%v", i)
}

// callsToAmputatorApi updates the calls to Amputator API value
// in both the local bot stats and in the database
func (bot *amputatorBot) updateCallsToAmputatorApi(i int) {
	field := getTagValueByTag("sql", "callsToAmputatorApi")
	localStats := bot.currentStats
	localStats.callsToAmputatorApi = i
	bot.statsChannel <- localStats
	if field == "" {
		log.Error("sql tag was blank for callsToAmputatorApi")
		return
	}
	bot.dbChannel <- field + " = " + fmt.Sprintf("%v", i)
}

// urlsAmputated updates the URLs amputated value
// in both the local bot stats and in the database
func (bot *amputatorBot) updateUrlsAmputated(i int) {
	field := getTagValueByTag("sql", "urlsAmputated")
	localStats := bot.currentStats
	localStats.urlsAmputated = i
	bot.statsChannel <- localStats
	if field == "" {
		log.Error("sql tag was blank for urlsAmputated")
		return
	}
	bot.dbChannel <- field + " = " + fmt.Sprintf("%v", i)
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

	field := getTagValueByTag("sql", "serversWatched")
	localStats := bot.currentStats
	localStats.serversWatched = i
	bot.statsChannel <- localStats
	if field == "" {
		log.Error("sql tag was blank for serversWatched")
		return
	}
	bot.dbChannel <- field + " = " + fmt.Sprintf("%v", i)
}
