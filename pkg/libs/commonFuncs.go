package libs

import (
	"encoding/json"
	"fmt"
	"github.com/alexbilevskiy/tgWatch/pkg/config"
	"io"
	"log"
	"os"
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

func jsonMarshalPretty(j interface{}) []byte {
	m, err := json.MarshalIndent(j, "", "    ")
	if err != nil {

		return []byte("INVALID_JSON")
	}

	return m
}

func FormatDateTime(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("2006-01-02 15:04:05")
}

func FormatDateTimeOs(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("2006-01-02_150405")
}

func FormatTime(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("15:04")
}

func FormatDate(timestamp int32) string {

	return time.Unix(int64(timestamp), 0).Format("2006-01-02")
}

func DLog(format string) {
	if config.Config.Debug {
		log.Printf(format)
	}
}

func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("Couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("Couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("Writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("Failed removing original file: %s", err)
	}
	return nil
}
