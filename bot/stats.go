package bot

import (
	"fmt"
)

type botStats struct {
	MessagesActedOn     int64  `pretty:"Messages Acted On"`
	MessagesSent        int64  `pretty:"Messages Sent"`
	CallsToAmputatorAPI int64  `pretty:"Calls to Amputator API"`
	URLsAmputated       int64  `pretty:"URLs Amputated"`
	TopDomains          string `pretty:"Top 5 Domains"`
	ServersWatched      int64  `pretty:"Servers Watched"`
}

type domainStats struct {
	ResponseDomainName string
	Count              int
}

// getGlobalStats calls the database to get global stats for the bot.
// The output here is not appropriate to send to individual servers, except
// for ServersWatched.
func (bot *AmputatorBot) getGlobalStats() botStats {
	var MessagesActedOn, MessagesSent, CallsToAmputatorAPI, ServersWatched int64
	serverId := bot.DG.State.User.ID
	amputationRows := []AmputationEvent{}
	var topDomains []domainStats

	bot.DB.Model(&MessageEvent{}).Count(&MessagesActedOn)
	bot.DB.Model(&MessageEvent{}).Where(&MessageEvent{AuthorId: serverId}).Count(&MessagesSent)
	bot.DB.Model(&Amputation{}).Where(&Amputation{Cached: false}).Count(&CallsToAmputatorAPI)
	bot.DB.Model(&Amputation{}).Scan(&amputationRows)
	bot.DB.Model(&Amputation{}).Select("response_domain_name, count(response_domain_name) as count").
		Group("response_domain_name").Order("count DESC").Find(&topDomains)
	bot.DB.Model(&ServerRegistration{}).Where(&ServerRegistration{}).Count(&ServersWatched)

	var topDomainsFormatted string
	for i := 0; i < 5 && i < len(topDomains); i++ {
		topDomainsFormatted = topDomainsFormatted + topDomains[i].ResponseDomainName + ": " +
			fmt.Sprintf("%v", topDomains[i].Count) + "\n"
	}

	if topDomainsFormatted == "" {
		topDomainsFormatted = "none"
	}

	return botStats{
		MessagesActedOn:     MessagesActedOn,
		MessagesSent:        MessagesSent,
		CallsToAmputatorAPI: CallsToAmputatorAPI,
		URLsAmputated:       int64(len(amputationRows)),
		TopDomains:          topDomainsFormatted,
		ServersWatched:      ServersWatched,
	}
}

// getServerStats gets the stats for a particular server with ID serverId.
// If you want global stats, use getGlobalStats()
func (bot *AmputatorBot) getServerStats(serverId string) botStats {
	var MessagesActedOn, MessagesSent, CallsToAmputatorAPI, ServersWatched int64
	botId := bot.DG.State.User.ID
	amputationRows := []AmputationEvent{}
	var topDomains []domainStats

	bot.DB.Model(&MessageEvent{}).Where(&MessageEvent{ServerID: serverId}).Count(&MessagesActedOn)
	bot.DB.Model(&MessageEvent{}).Where(&MessageEvent{AuthorId: botId, ServerID: serverId}).Count(&MessagesSent)
	bot.DB.Model(&Amputation{}).Where(&Amputation{ServerID: serverId, Cached: false}).Count(&CallsToAmputatorAPI)
	bot.DB.Model(&Amputation{}).Where(&Amputation{ServerID: serverId}).Scan(&amputationRows)
	bot.DB.Model(&Amputation{}).Where(&Amputation{ServerID: serverId}).
		Select("response_domain_name, count(response_domain_name) as count").Order("count DESC").
		Group("response_domain_name").Find(&topDomains)
	bot.DB.Model(&ServerRegistration{}).Where(&ServerRegistration{}).Count(&ServersWatched)

	var topDomainsFormatted string
	for i := 0; i < 5 && i < len(topDomains); i++ {
		topDomainsFormatted = topDomainsFormatted + topDomains[i].ResponseDomainName + ": " +
			fmt.Sprintf("%v", topDomains[i].Count) + "\n"
	}

	if topDomainsFormatted == "" {
		topDomainsFormatted = "none"
	}

	return botStats{
		MessagesActedOn:     MessagesActedOn,
		MessagesSent:        MessagesSent,
		CallsToAmputatorAPI: CallsToAmputatorAPI,
		URLsAmputated:       int64(len(amputationRows)),
		TopDomains:          topDomainsFormatted,
		ServersWatched:      ServersWatched,
	}
}
