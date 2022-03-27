package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

func (bot amputatorBot) updateMessagesSeen(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET messagesSeen = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update messagesSeen: ", err)
		}
	}
}

func (bot amputatorBot) updateMessagesActedOn(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET messagesActedOn = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update messagesActedOn: ", err)
		}
	}
}

func (bot amputatorBot) updateMessagesSent(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET messagesSent = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update messagesSent: ", err)
		}
	}
}

func (bot amputatorBot) updateCallsToAmputatorApi(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET callsToAmputatorApi = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan(bot.stats)
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update callsToAmputatorApi: ", err)
		}
	}
}

func (bot amputatorBot) updateUrlsAmputated(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET urlsAmputated = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan(bot.stats)
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update urlsAmputated: ", err)
		}
	}
}

func (bot amputatorBot) updateServersWatched(s *discordgo.Session, serverCount int) {
	logrus.Info("watching ", serverCount, " servers")
	usd := &discordgo.UpdateStatusData{Status: "online"}
	usd.Activities = make([]*discordgo.Activity, 1)
	usd.Activities[0] = &discordgo.Activity{
		Name: fmt.Sprintf("%v servers", serverCount),
		Type: discordgo.ActivityTypeWatching,
		URL:  "https://github.com/tyzbit/go-discord-amputator",
	}

	err := s.UpdateStatusComplex(*usd)
	if err != nil {
		logrus.Error("failed to set status: ", err)
	}

	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET serversWatched = " + fmt.Sprint(serverCount) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update serversWatched: ", err)
		}
	}
}

func (bot amputatorBot) updateOrInitializeBotStats() (amputatorBot, error) {
	if !bot.dbConnected {
		logrus.Debug("not updating db stats because DB is not connected")
		return bot, nil
	}

	var botId, messagesSeen, messagesActedOn, messagesSent, callsToAmputatorApi, urlsAmputated, serversWatched int
	getStatsQuery := "SELECT * FROM stats WHERE botId = " + fmt.Sprintf("%v", bot.id)
	query := bot.dbConnection.QueryRow(getStatsQuery)
	err := query.Scan(&botId, &callsToAmputatorApi, &urlsAmputated, &serversWatched, &messagesSeen, &messagesActedOn, &messagesSent)
	if err != nil {
		logrus.Debug("unable to pull stats from database, err: ", err, ", creating table if not exists")
		createTableQuery := "CREATE TABLE IF NOT EXISTS stats (botId int PRIMARY KEY, "
		statsTypes := []string{}
		for field, value := range bot.stats {
			statsTypes = append(statsTypes, fmt.Sprintf("%v %T DEFAULT 0", field, value))
		}
		createTableQuery = createTableQuery + strings.Join(statsTypes, ", ") + ");"
		query := bot.dbConnection.QueryRow(createTableQuery)
		err = query.Scan()
		if err != sql.ErrNoRows {
			return bot, fmt.Errorf("error creating table: %v", err)
		}

		logrus.Debug("initializing values in database")
		statsValues := []string{}
		statsColumns := []string{}
		for field, value := range bot.stats {
			statsColumns = append(statsColumns, field)
			statsValues = append(statsValues, fmt.Sprintf("%v", value))
		}
		createRecordQuery := "INSERT INTO stats (botId, " + strings.Join(statsColumns, ", ") + ") VALUE (" + fmt.Sprintf("%v", bot.id) + ", " + strings.Join(statsValues, ", ") + ");"
		logrus.Debug("initializing query: ", createRecordQuery)
		query = bot.dbConnection.QueryRow(createRecordQuery)
		err = query.Scan()
		if err != sql.ErrNoRows {
			return bot, fmt.Errorf("unable to add initial stats row: %v", err)
		}

		query = bot.dbConnection.QueryRow(getStatsQuery)
		err := query.Scan(&botId, &messagesSeen, &messagesActedOn, &messagesSent, &callsToAmputatorApi, &urlsAmputated, &serversWatched)
		if err != nil {
			bot.dbConnected = false
			return bot, fmt.Errorf("unable to get just-created stats: %v", err)
		}
	}

	bot.stats = map[string]int{
		"messagesSeen":        messagesSeen,
		"messagesActedOn":     messagesActedOn,
		"messagesSent":        messagesSent,
		"callsToAmputatorApi": callsToAmputatorApi,
		"urlsAmputated":       urlsAmputated,
		"serversWatched":      serversWatched,
	}
	logrus.Debug("successfully updated stats: ", fmt.Sprintf("%v", bot.stats))
	return bot, nil
}
