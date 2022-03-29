package main

import (
	"reflect"
)

// getTagValueByTag gets the tag value for a given tag name on a type
// such as struct.
func getTagValueByTag(tag string, name string) string {
	s := reflect.TypeOf(amputatorStats{})
	f, _ := s.FieldByName(name)
	v, ok := f.Tag.Lookup(tag)
	if ok {
		return v
	}
	return ""
}
