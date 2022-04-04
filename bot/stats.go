package bot

type botStats struct {
	MessagesActedOn     int64 `pretty:"Messages Seen"`
	MessagesSent        int64 `pretty:"Messages Sent"`
	CallsToAmputatorAPI int64 `pretty:"Calls to Amputator API"`
	URLsAmputated       int64 `pretty:"URLs Amputated"`
	ServersWatched      int64 `pretty:"Servers Watched"`
}

// getGlobalStats calls the database to get global stats for the bot.
// The output here is not appropriate to send to individual servers, except
// for ServersWatched.
func (bot *AmputatorBot) getGlobalStats() botStats {
	var MessagesActedOn, MessagesSent, CallsToAmputatorAPI, ServersWatched int64
	serverId := bot.DG.State.User.ID
	amputationRows := []AmputationEvent{}

	bot.DB.Model(&MessageEvent{}).Count(&MessagesActedOn)
	bot.DB.Model(&MessageEvent{}).Where(&MessageEvent{AuthorId: serverId}).Count(&MessagesSent)
	bot.DB.Model(&AmputationEvent{}).Count(&CallsToAmputatorAPI)
	bot.DB.Model(&AmputationEvent{}).Scan(&amputationRows)
	bot.DB.Model(&ServerRegistration{}).Where(&ServerRegistration{}).Count(&ServersWatched)

	return botStats{
		MessagesActedOn:     MessagesActedOn,
		MessagesSent:        MessagesSent,
		CallsToAmputatorAPI: CallsToAmputatorAPI,
		URLsAmputated:       int64(len(amputationRows)),
		ServersWatched:      ServersWatched,
	}
}

// getServerStats gets the stats for a particular server with ID serverId.
// If you want global stats, use getGlobalStats()
func (bot *AmputatorBot) getServerStats(serverId string) botStats {
	var MessagesActedOn, MessagesSent, AmputationEvents, ServersWatched int64
	botId := bot.DG.State.User.ID
	amputationRows := []AmputationEvent{}

	bot.DB.Model(&MessageEvent{}).Where(&MessageEvent{ServerId: serverId}).Count(&MessagesActedOn)
	bot.DB.Model(&MessageEvent{}).Where(&MessageEvent{AuthorId: botId, ServerId: serverId}).Count(&MessagesSent)
	bot.DB.Model(&AmputationEvent{}).Where(&AmputationEvent{ServerId: serverId}).Count(&AmputationEvents)
	bot.DB.Model(&AmputationEvent{}).Where(&AmputationEvent{ServerId: serverId}).Scan(&amputationRows)
	bot.DB.Model(&ServerRegistration{}).Where(&ServerRegistration{}).Count(&ServersWatched)

	return botStats{
		MessagesActedOn,
		MessagesSent,
		AmputationEvents,
		int64(len(amputationRows)),
		ServersWatched,
	}
}
