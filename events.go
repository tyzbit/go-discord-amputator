package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// A messageEvent is created when we receive a message that
// requires our attention
type messageEvent struct {
	gorm.Model
	UUID             string `gorm:"primaryKey"`
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
	gorm.Model
	UUID           string `gorm:"primaryKey"`
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
	gorm.Model
	UUID                string `gorm:"primaryKey"`
	AmputationEventUUID string
	Type                string
	URL                 string
	DomainName          string
}

// createMessageEvent logs a given message event into the database and
// returns the UUID for the message.
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
