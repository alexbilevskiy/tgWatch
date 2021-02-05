package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func ImplodeInt(a []int64) string {

	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", ",", -1), "[]")
}

func ExplodeInt(a string) []int64 {
	stringsArr := strings.Split(a, ",")
	var intsArr []int64
	for _, v := range stringsArr {
		one, _ := strconv.ParseInt(v, 10, 64)
		intsArr = append(intsArr, one)
	}
	return intsArr
}


func JsonMarshalStr(j interface{}) string {
	m, err := json.Marshal(j)
	if err != nil {

		return "INVALID_JSON"
	}

	return string(m)
}
