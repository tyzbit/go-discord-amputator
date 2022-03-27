package main

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

// updateMessagesSeen updates the count of the number of messages seen in the database.
func (bot amputatorBot) updateMessagesSeen(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET messagesSeen = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update messagesSeen: ", err)
		}
	}
}

// updateMessagesActedOn updates the count of the number of messages acted on
// in the database.
func (bot amputatorBot) updateMessagesActedOn(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET messagesActedOn = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update messagesActedOn: ", err)
		}
	}
}

// updateMessagesSent updates the count of the number of messages sent in the database.
func (bot amputatorBot) updateMessagesSent(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET messagesSent = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update messagesSent: ", err)
		}
	}
}

// updateMessagesSent updates the count of the number of calls to the Amputator API in the database.
func (bot amputatorBot) updateCallsToAmputatorApi(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET callsToAmputatorApi = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update callsToAmputatorApi: ", err)
		}
	}
}

// updateUrlsAmputated updates the count of the number of URLs amputated in the database.
func (bot amputatorBot) updateUrlsAmputated(i int) {
	if bot.dbConnected {
		query := bot.dbConnection.QueryRow("UPDATE stats SET urlsAmputated = " + fmt.Sprint(i) + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";")
		err := query.Scan()
		if err != sql.ErrNoRows {
			logrus.Warn("unable to update urlsAmputated: ", err)
		}
	}
}

// updateServersWatched updates the number of servers watched internally and
// also updates the value in the database
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

// updateOrInitializeBotStats loads bot stats from the database. If not found,
// it creates a stats table with an initial entry.
func (bot amputatorBot) updateOrInitializeBotStats() (amputatorBot, error) {
	if !bot.dbConnected {
		logrus.Debug("not updating db stats because DB is not connected")
		return bot, nil
	}

	var botId, callsToAmputatorApi, messagesActedOn, messagesSeen, messagesSent, serversWatched, urlsAmputated int
	getStatsQuery := "SELECT * FROM stats WHERE botId = " + fmt.Sprintf("%v", bot.id)
	query := bot.dbConnection.QueryRow(getStatsQuery)
	err := query.Scan(&botId, &callsToAmputatorApi, &messagesActedOn, &messagesSeen, &messagesSent, &serversWatched, &urlsAmputated)
	if err != nil {
		logrus.Debug("unable to pull stats from database, err: ", err, ", creating table if not exists")
		createTableQuery := "CREATE TABLE IF NOT EXISTS stats (botId int PRIMARY KEY, "
		statsTypes := []string{}

		// Sort keys first to ensure order is preserved
		keys := make([]string, 0, len(bot.stats))
		for k := range bot.stats {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			statsTypes = append(statsTypes, fmt.Sprintf("%v %T DEFAULT 0", k, bot.stats[k]))

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

		// Sort keys first to ensure order is preserved
		keys = make([]string, 0, len(bot.stats))
		for k := range bot.stats {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			statsColumns = append(statsColumns, k)
			statsValues = append(statsValues, fmt.Sprintf("%v", bot.stats[k]))
		}
		createRecordQuery := "INSERT INTO stats (botId, " + strings.Join(statsColumns, ", ") + ") VALUE (" + fmt.Sprintf("%v", bot.id) + ", " + strings.Join(statsValues, ", ") + ");"
		logrus.Debug("initializing query: ", createRecordQuery)
		query = bot.dbConnection.QueryRow(createRecordQuery)
		err = query.Scan()
		if err != sql.ErrNoRows {
			return bot, fmt.Errorf("unable to add initial stats row: %v", err)
		}

		query = bot.dbConnection.QueryRow(getStatsQuery)
		err := query.Scan(&botId, &callsToAmputatorApi, &messagesActedOn, &messagesSeen, &messagesSent, &serversWatched, &urlsAmputated)
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
