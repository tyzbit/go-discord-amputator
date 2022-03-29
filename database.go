package main

import (
	"database/sql"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

// writeStatToDatabase writes a particular stat to the database.
// it takes "KEY = VALUE" which is used in the SQL UPDATE statement.
func (bot *amputatorBot) updateValueInDb(queryFragment string) error {
	if bot.dbConnected {
		statement := "UPDATE stats SET " + queryFragment + " WHERE botId = " + fmt.Sprintf("%v", bot.id) + ";"
		log.Trace("updateValueInDb query: ", statement)
		query := bot.dbConnection.QueryRow(statement)
		err := query.Scan()
		if err != sql.ErrNoRows {
			log.Warn("unable to run query (", queryFragment, "): ", err)
			return err
		}
	}
	return nil
}

// initializeBotStats loads bot stats from the database. If not found,
// it creates a stats table with an initial entry.
func (bot *amputatorBot) initializeBotStats() (amputatorBot, error) {
	if !bot.dbConnected {
		log.Debug("not updating db stats because DB is not connected")
		return *bot, nil
	}

	// Try to get stats from the database. If unsuccessful, try creating the table,
	// then initialize the stats in the database.
	getStatsQuery := "SELECT * FROM stats WHERE botId = " + fmt.Sprintf("%v", bot.id)
	log.Trace("getting stats from db with query: ", getStatsQuery)
	query := bot.dbConnection.QueryRow(getStatsQuery)
	err := query.Scan(&bot.id,
		&bot.currentStats.messagesSeen,
		&bot.currentStats.messagesActedOn,
		&bot.currentStats.messagesSent,
		&bot.currentStats.callsToAmputatorApi,
		&bot.currentStats.urlsAmputated,
		&bot.currentStats.serversWatched,
	)
	if err != nil {
		log.Info("unable to pull stats from database, err: ", err, ", creating table if not exists")
		createTableQuery := "CREATE TABLE IF NOT EXISTS stats ("

		statsList := []string{
			"messagesSeen",
			"messagesActedOn",
			"messagesSent",
			"callsToAmputatorApi",
			"urlsAmputated",
			"serversWatched",
		}

		statsTypes := []string{}
		statsTypes = append(statsTypes, fmt.Sprintf("botId int PRIMARY KEY"))
		for _, stat := range statsList {
			statsTypes = append(statsTypes, fmt.Sprintf("%v int DEFAULT 0", getTagValueByTag("sql", stat)))
		}

		createTableQuery = createTableQuery + strings.Join(statsTypes, ", ") + ");"
		log.Trace("creating table in db with query: ", createTableQuery)
		query := bot.dbConnection.QueryRow(createTableQuery)
		err = query.Scan()
		if err != sql.ErrNoRows {
			return *bot, fmt.Errorf("error creating table: %v", err)
		}

		log.Info("initializing values in database")
		statsColumns := []string{fmt.Sprintf("botId")}
		statsValues := []string{fmt.Sprintf("%v", bot.id)}
		for _, stat := range statsList {
			statsColumns = append(statsColumns, getTagValueByTag("sql", stat))
			statsValues = append(statsValues, "0")
		}

		createRecordQuery := "INSERT INTO stats (" + strings.Join(statsColumns, ", ") + ") VALUE (" + strings.Join(statsValues, ", ") + ");"
		log.Trace("inserting initial stats into db with query: ", createRecordQuery)
		query = bot.dbConnection.QueryRow(createRecordQuery)
		err = query.Scan()
		if err != sql.ErrNoRows {
			return *bot, fmt.Errorf("unable to add initial stats row: %v", err)
		}

		query = bot.dbConnection.QueryRow(getStatsQuery)
		err := query.Scan(&bot.id,
			&bot.currentStats.messagesSeen,
			&bot.currentStats.messagesActedOn,
			&bot.currentStats.messagesSent,
			&bot.currentStats.callsToAmputatorApi,
			&bot.currentStats.urlsAmputated,
			&bot.currentStats.serversWatched,
		)
		if err != nil {
			bot.dbConnected = false
			return *bot, fmt.Errorf("unable to get just-created stats: %v", err)
		}
	}

	log.Info("successfully initialized stats from the database")
	log.Trace("stats: ", fmt.Sprintf("%v", bot.currentStats))
	return *bot, nil
}
