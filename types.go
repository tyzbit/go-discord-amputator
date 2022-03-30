package main

import (
	"sync"

	"github.com/jmoiron/sqlx"
)

type amputatorBot struct {
	db          *sqlx.DB
	dbConnected bool
	dbUpdates   chan string
	info        botInfo
	infoUpdates sync.Mutex
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

type botInfo struct {
	ID                  int `db:"botId" pretty:"Bot ID"`
	MessagesSeen        int `db:"messagesSeen" pretty:"Messages Seen"`
	MessagesActedOn     int `db:"messagesActedOn" pretty:"Messages Acted On"`
	MessagesSent        int `db:"messagesSent" pretty:"Messages Sent"`
	CallsToAmputatorAPI int `db:"callsToAmputatorApi" pretty:"Calls to Amputator API"`
	URLsAmputated       int `db:"urlsAmputated" pretty:"URLs Amputated"`
	ServersWatched      int `db:"serversWatched" pretty:"Servers Watched"`
}
