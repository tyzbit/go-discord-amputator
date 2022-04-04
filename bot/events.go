package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// A MessageEvent is created when we receive a message that
// requires our attention
type MessageEvent struct {
	UUID             string `gorm:"primaryKey"`
	CreatedAt        time.Time
	AuthorId         string
	AuthorUsername   string
	MessageId        string
	Command          string
	ChannelId        string
	ServerId         string
	AmputationEvents []AmputationEvent `gorm:"foreignKey:UUID"`
}

// Every successful AmputationEvent will come from a message.
type AmputationEvent struct {
	UUID           string `gorm:"primaryKey"`
	CreatedAt      time.Time
	AuthorId       string
	AuthorUsername string
	ChannelId      string
	MessageId      string
	ServerId       string
	RequestURLs    []URLInfo `gorm:"foreignKey:AmputationEventUUID"`
	ResponseURLs   []URLInfo `gorm:"foreignKey:AmputationEventUUID"`
}

// This is the representation of request and response URLs from users or
// the Amputator API.
type URLInfo struct {
	UUID                string `gorm:"primaryKey"`
	CreatedAt           time.Time
	AmputationEventUUID string
	Type                string
	URL                 string
	DomainName          string
}

// createMessageEvent logs a given message event into the database.
func (bot *AmputatorBot) createMessageEvent(c string, m *discordgo.Message) {
	uuid := uuid.New().String()
	bot.DB.Create(&MessageEvent{
		UUID:           uuid,
		AuthorId:       m.Author.ID,
		AuthorUsername: m.Author.Username,
		MessageId:      m.ID,
		Command:        c,
		ChannelId:      m.ChannelID,
		ServerId:       m.GuildID,
	})
}
