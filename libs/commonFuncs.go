package libs

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"tgWatch/config"
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

func jsonMarshalPretty(j interface{}) string {
	m, err := json.MarshalIndent(j, "", "    ")
	if err != nil {

		return "INVALID_JSON"
	}

	return string(m)
}

func FormatDateTime(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("2006-01-02 15:04:05")
}

func FormatTime(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("15:04")
}

func FormatDate(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("2006-01-02")
}

func DLog(format string) {
	if config.Config.Debug {
		log.Print(format)
	}
}
func NDLog(format string) {
	log.Print(format)
}
