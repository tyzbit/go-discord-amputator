package main

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// A messageEvent is created when we receive a message that
// requires our attention
type messageEvent struct {
	UUID             string `gorm:"primaryKey"`
	CreatedAt        time.Time
	AuthorId         string
	AuthorUsername   string
	MessageId        string
	Command          string
	ChannelId        string
	ServerId         string
	AmputationEvents []amputationEvent `gorm:"foreignKey:UUID"`
}

// Every successful amputationEvent will come from a message.
type amputationEvent struct {
	UUID           string `gorm:"primaryKey"`
	CreatedAt      time.Time
	AuthorId       string
	AuthorUsername string
	ChannelId      string
	MessageId      string
	ServerId       string
	RequestURLs    []urlInfo `gorm:"foreignKey:AmputationEventUUID"`
	ResponseURLs   []urlInfo `gorm:"foreignKey:AmputationEventUUID"`
}

// This is the representation of request and response URLs from users or
// the Amputator API.
type urlInfo struct {
	UUID                string `gorm:"primaryKey"`
	CreatedAt           time.Time
	AmputationEventUUID string
	Type                string
	URL                 string
	DomainName          string
}

// createMessageEvent logs a given message event into the database.
func (bot *amputatorBot) createMessageEvent(c string, m *discordgo.Message) {
	uuid := uuid.New().String()
	bot.db.Create(&messageEvent{
		UUID:           uuid,
		AuthorId:       m.Author.ID,
		AuthorUsername: m.Author.Username,
		MessageId:      m.ID,
		Command:        c,
		ChannelId:      m.ChannelID,
		ServerId:       m.GuildID,
	})
}