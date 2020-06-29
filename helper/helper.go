package helper

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"
)

func PrettyPrint(i interface{}) string {
	s, err := json.MarshalIndent(i, "", "\t")
	if err != nil {
		log.Printf("prettprint.Get err   #%v ", err)
	}
	return string(s)
}

func Snakeify(data string) string {
	space := regexp.MustCompile(`\s+`)
	data = strings.ToLower(data)
	data = space.ReplaceAllString(data, "_")
	return data
}

func Kebabify(data string) string {
	space := regexp.MustCompile(`(\s|_)+`)
	data = strings.ToLower(data)
	data = space.ReplaceAllString(data, "-")
	return data
}

func Pascalify(data string) string {
	space := regexp.MustCompile(`\s+`)
	data = strings.Title(data)
	data = space.ReplaceAllString(data, "")
	return data
}
