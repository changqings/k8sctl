package utils

import (
	"strings"
)

func StringToMap(str string) map[string]string {

	var tempMap = make(map[string]string)

	tmpSlice := strings.Split(str, ",")
	for _, s := range tmpSlice {
		m := strings.Split(s, "=")
		tempMap[m[0]] = m[1]
	}

	return tempMap
}
