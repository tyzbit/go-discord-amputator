package main

import (
	"reflect"
)

// getBotInfoTagValue gets the tag value for a given tag name on a type
// such as struct.
func getBotInfoTagValue(tag string, name string) string {
	s := reflect.TypeOf(botInfo{})
	f, _ := s.FieldByName(name)
	v, ok := f.Tag.Lookup(tag)
	if ok {
		return v
	}
	return ""
}
