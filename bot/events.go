package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// A MessageEvent is created when we receive a message that
// requires our attention
type MessageEvent struct {
	CreatedAt        time.Time
	UUID             string `gorm:"primaryKey"`
	AuthorId         string
	AuthorUsername   string
	MessageId        string
	Command          string
	ChannelId        string
	ServerID         string
	AmputationEvents []AmputationEvent `gorm:"foreignKey:UUID"`
}

// Every successful AmputationEvent will come from a message.
type AmputationEvent struct {
	CreatedAt      time.Time
	UUID           string `gorm:"primaryKey"`
	AuthorId       string
	AuthorUsername string
	ChannelId      string
	MessageId      string
	ServerID       string
	Amputations    []Amputation `gorm:"foreignKey:AmputationEventUUID"`
}

// This is the representation of request and response URLs from users or
// the Amputator API.
type Amputation struct {
	CreatedAt           time.Time
	UUID                string `gorm:"primaryKey"`
	AmputationEventUUID string
	ServerID            string
	RequestURL          string
	RequestDomainName   string
	ResponseURL         string
	ResponseDomainName  string
	Cached              bool
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
		ServerID:       m.GuildID,
	})
}
