package main

type botStats struct {
	MessagesActedOn     int64 `pretty:"Messages Seen"`
	MessagesSent        int64 `pretty:"Messages Sent"`
	CallsToAmputatorAPI int64 `pretty:"Calls to Amputator API"`
	URLsAmputated       int64 `pretty:"URLs Amputated"`
	ServersWatched      int64 `pretty:"Servers Watched"`
}

func (bot *amputatorBot) getGlobalStats() botStats {
	var MessagesActedOn, MessagesSent, AmputationEvents, ServersWatched int64
	serverId := bot.dg.State.User.ID
	amputationRows := []amputationEvent{}

	bot.db.Model(&messageEvent{}).Count(&MessagesActedOn)
	bot.db.Model(&messageEvent{}).Where(&messageEvent{AuthorId: serverId}).Count(&MessagesSent)
	bot.db.Model(&amputationEvent{}).Count(&AmputationEvents)
	bot.db.Model(&amputationEvent{}).Scan(&amputationRows)
	bot.db.Model(&serverRegistration{}).Where(&serverRegistration{}).Count(&ServersWatched)

	return botStats{
		MessagesActedOn,
		MessagesSent,
		AmputationEvents,
		int64(len(amputationRows)),
		ServersWatched,
	}
}

func (bot *amputatorBot) getServerStats(serverId string) botStats {
	var MessagesActedOn, MessagesSent, AmputationEvents, ServersWatched int64
	botId := bot.dg.State.User.ID
	amputationRows := []amputationEvent{}

	bot.db.Model(&messageEvent{}).Where(&messageEvent{ServerId: serverId}).Count(&MessagesActedOn)
	bot.db.Model(&messageEvent{}).Where(&messageEvent{AuthorId: botId, ServerId: serverId}).Count(&MessagesSent)
	bot.db.Model(&amputationEvent{}).Where(&amputationEvent{ServerId: serverId}).Count(&AmputationEvents)
	bot.db.Model(&amputationEvent{}).Where(&amputationEvent{ServerId: serverId}).Scan(&amputationRows)
	bot.db.Model(&serverRegistration{}).Where(&serverRegistration{}).Count(&ServersWatched)

	return botStats{
		MessagesActedOn,
		MessagesSent,
		AmputationEvents,
		int64(len(amputationRows)),
		ServersWatched,
	}
}
