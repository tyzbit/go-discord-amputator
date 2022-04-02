package main

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type interfaceMap map[string]interface{}

// getFieldNamesByType takes an interface as an argument
// and returns an array of the field names
func convertFlatStructToSliceStringMap(i interface{}) []map[string]string {
	// Get reflections
	t := reflect.TypeOf(i)
	t2 := reflect.ValueOf(i)

	values := map[string]string{}
	// Keys is a list of keys of the values map. It's used for sorting later
	keys := make([]string, 0, len(values))

	// Convert the struct to map[string]string
	for i := 0; i < t.NumField(); i++ {
		k := t.Field(i).Name
		v := t2.Field(i)
		values[k] = fmt.Sprintf("%v", v)
		keys = append(keys, k)
	}

	//
	sort.Strings(keys)

	sortedValues := make([]map[string]string, 0, t.NumField())
	for _, k := range keys {
		sortedValues = append(sortedValues, map[string]string{k: values[k]})
	}

	return sortedValues
}

// getTagValue looks up the tag for a given field of the specified type.
func getTagValue(i interface{}, field string, tag string) string {
	r, ok := reflect.TypeOf(i).FieldByName(field)
	if !ok {
		return ""
	}
	return r.Tag.Get(tag)
}

// Returns a multiline string that pretty prints botStats. Ripped from
// https://stackoverflow.com/a/18927729
func structToDiscordFields(i any) []*discordgo.MessageEmbedField {
	var fields ([]*discordgo.MessageEmbedField)

	stringMapSlice := convertFlatStructToSliceStringMap(i)

	for _, stringMap := range stringMapSlice {
		for key, value := range stringMap {
			formattedKey := getTagValue(i, key, "pretty")
			newField := discordgo.MessageEmbedField{
				Name:  formattedKey,
				Value: fmt.Sprintf("%v", value),
			}
			fields = append(fields, &newField)
		}
	}

	return fields
}

// sendMessage sends a MessageEmbed or a regular message. The content of the regular
// message is the description of the passed MessageEmbed
func (b amputatorBot) sendMessage(s *discordgo.Session, useEmbed bool, replyTo bool,
	m *discordgo.Message, e *discordgo.MessageEmbed) {

	var err error
	switch useEmbed {
	case true:
		_, err = s.ChannelMessageSendEmbed(m.ChannelID, e)
		if err != nil {
			log.Warn(err)
		}
	case false:
		if !replyTo {
			_, err = s.ChannelMessageSend(m.ChannelID, e.Description)
		} else {
			_, err = s.ChannelMessageSendReply(m.ChannelID, e.Description, m.Reference())
		}
		if err != nil {
			log.Warn(err)
		}
	}
}

// getDomainName receives a URL and returns the FQDN
func getDomainName(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		log.Error("unable to determine domain name for url: ", s)
	}
	parts := strings.Split(u.Hostname(), ".")
	return parts[len(parts)-2] + "." + parts[len(parts)-1]
}
