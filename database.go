package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Watches the dbUpdates channel and runs DB queries
func (bot *amputatorBot) dbHandler() {
	for queryFragment := range bot.dbUpdates {
		if bot.dbConnected {
			err := bot.updateValueInDb(queryFragment)
			if err != nil {
				log.Error("error updating value in db: ", err)
			}
		}
	}
}

// writeStatToDatabase writes a particular stat to the database.
// it takes "KEY = VALUE" which is used in the SQL UPDATE statement.
func (bot *amputatorBot) updateValueInDb(queryFragment string) error {
	statement := "UPDATE stats SET " + queryFragment +
		" WHERE " + getBotInfoTagValue("db", "ID") + " = " + fmt.Sprintf("%v", bot.info.ID) + ";"

	log.Trace("updateValueInDb query(", statement, ")")
	result, err := bot.db.Exec(statement)
	if err != nil {
		log.Warn("unable to run query(", queryFragment, "): ", err)
		return err
	}

	r, err := result.RowsAffected()
	if err != nil {
		log.Warn("unable to determine rows affected, query(", queryFragment, "): ", err)
		return err
	}
	log.Trace(r, " rows affected")
	return nil
}

// createBot returns a bot with stats from the database. If not found,
// it creates a stats table with an initial entry.
func (bot *amputatorBot) initializeStats() {
	if !bot.dbConnected {
		log.Debug("not updating db stats because DB is not connected")
		return
	}

	// Try to get stats from the database. If unsuccessful, try creating the table,
	// then initialize the stats in the database.
	getStatsQuery := "SELECT * FROM stats WHERE botId = " + fmt.Sprintf("%v", bot.info.ID) + " LIMIT 1;"
	log.Trace("getting stats from db with query(", getStatsQuery, ")")
	statsRows := []botInfo{}
	err := bot.db.Select(&statsRows, getStatsQuery)
	if err != nil {
		log.Info("unable to load stats from database, err: ", err, ", trying to initialize stats table")
		statsRows = bot.initializeDbStats(statsRows, getStatsQuery)
		return
	}

	if len(statsRows) == 0 {
		log.Info("no stats rows returned: ", err, ", trying to initialize stats table")
		statsRows = bot.initializeDbStats(statsRows, getStatsQuery)
		return
	}

	if len(statsRows) > 1 {
		log.Info("too many stats rows returned, stats will be from 0: ", err)
		bot.dbConnected = false
		return
	}

	// Save bot info from database
	bot.info = statsRows[0]

	log.Info("successfully initialized stats from the database")
	log.Trace("stats: ", fmt.Sprintf("%#v", &bot.info))
	return
}

func (bot *amputatorBot) initializeDbStats(statsRows []botInfo, getStatsQuery string) []botInfo {
	// Update this if the botInfo struct changes
	createTableQuery := "CREATE TABLE IF NOT EXISTS stats (" +
		"botId int PRIMARY KEY," +
		"messagesSeen int DEFAULT 0," +
		"messagesActedOn int DEFAULT 0," +
		"messagesSent int DEFAULT 0," +
		"callsToAmputatorApi int DEFAULT 0," +
		"urlsAmputated int DEFAULT 0," +
		"serversWatched int DEFAULT 0" +
		")"

	log.Trace("creating table in db with query(", createTableQuery, ")")
	_, err := bot.db.Exec(createTableQuery)
	if err != nil {
		bot.dbConnected = false
		log.Errorf("error creating table: %v", err)
	}

	// Update this if the botInfo struct changes
	createRecordQuery := "INSERT INTO stats " +
		"(botId, messagesSeen, messagesActedOn, messagesSent, callsToAmputatorApi, urlsAmputated, serversWatched) " +
		"VALUES (" + fmt.Sprintf("%v", bot.info.ID) + ", 0, 0, 0, 0, 0, 0);"
	log.Trace("inserting initial stats into db with query(", createRecordQuery, ")")
	_, err = bot.db.Exec(createRecordQuery)
	if err != nil {
		bot.dbConnected = false
		log.Errorf("unable to add initial stats row: %v", err)
	}

	err = bot.db.Select(&statsRows, getStatsQuery)
	if err != nil {
		bot.dbConnected = false
		log.Errorf("unable to get just-created stats")
	}
	return statsRows
}
