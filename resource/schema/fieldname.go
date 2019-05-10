package schema

import (
	"reflect"
	"regexp"
	"strings"
)

var reFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var reAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func fieldName(f reflect.StructField) string {
	if n, ok := f.Tag.Lookup("name"); ok {
		return n
	}
	snake := reFirstCap.ReplaceAllString(f.Name, "${1}_${2}")
	snake = reAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
