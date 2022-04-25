package bot

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	allSchemaTypes = []interface{}{
		&ServerRegistration{},
		&ServerConfig{},
		&Amputation{},
		&AmputationEvent{},
		&MessageEvent{},
	}
)

func testInit() AmputatorBot {
	db, err := gorm.Open(sqlite.Open("./test.sqlite"))
	if err != nil {
		log.Error("unable to set up db: ", err)
	}

	// Set up DB if necessary
	for _, schemaType := range allSchemaTypes {
		err := db.AutoMigrate(schemaType)
		if err != nil {
			log.Fatal("unable to automigrate ", reflect.TypeOf(&schemaType).Elem().Name(), "err: ", err)
		}
	}

	return AmputatorBot{
		DB:         db,
		DG:         &discordgo.Session{StateEnabled: true, State: discordgo.NewState()},
		StartingUp: true,
	}
}

func TestBotReady(t *testing.T) {
	ampBot := testInit()
	_ = ampBot.DG.State.GuildAdd(&discordgo.Guild{ID: "guild"})
	_ = ampBot.DG.State.MemberAdd(&discordgo.Member{
		User: &discordgo.User{
			ID:       "user",
			Username: "User Name",
		},
		Nick:    "User Nick",
		GuildID: "guild",
	})
	ampBot.BotReady(ampBot.DG, &ampBot.DG.State.Ready)
}
