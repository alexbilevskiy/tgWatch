package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
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

func FormatTime(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("2006-01-02 15:04:05")
}