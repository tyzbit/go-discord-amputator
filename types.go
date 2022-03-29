package main

import (
	"database/sql"
)

type amputatorBot struct {
	id           int
	dbChannel    chan string
	dbConnected  bool
	dbConnection *sql.DB
	currentStats amputatorStats
	statsChannel chan amputatorStats
}

type amputatorBotConfig struct {
	adminIds              []string `env:"ADMINISTRATOR_IDS"`
	automaticallyAmputate bool     `env:"AUTOMATICALLY_AMPUTATE"`
	botId                 int      `env:"BOT_ID"`
	dbHost                string   `env:"DB_HOST"`
	dbName                string   `env:"DB_NAME"`
	dbPassword            string   `env:"DB_PASSWORD"`
	dbUser                string   `env:"DB_USER"`
	guessAndCheck         string   `env:"GUESS_AND_CHECK"`
	logLevel              string   `env:"LOG_LEVEL"`
	maxDepth              string   `env:"MAX_DEPTH"`
	token                 string   `env:"TOKEN"`
}

type amputatorStats struct {
	messagesSeen        int `sql:"messagesSeen" pretty:"Messages Seen"`
	messagesActedOn     int `sql:"messagesActedOn" pretty:"Messages Acted On"`
	messagesSent        int `sql:"messagesSent" pretty:"Messages Sent"`
	callsToAmputatorApi int `sql:"callsToAmputatorApi" pretty:"Calls to Amputator API"`
	urlsAmputated       int `sql:"urlsAmputated" pretty:"URLs Amputated"`
	serversWatched      int `sql:"serversWatched" pretty:"Servers Watched"`
}
