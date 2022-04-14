package libs

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"tgWatch/modules/vosk"
)

//CGO_CFLAGS= "-I/opt/src/vosk-api/src" CGO_LDFLAGS = "-L/opt/src/kaldi/tools/openfst/src/lib/.libs -L/opt/src/kaldi/tools/OpenBLAS/install/lib -L/opt/src/kaldi/tools/openfst/lib -L/opt/src/kaldi/tools/openfst/lib/fst -L/opt/src/vosk-api/src -lvosk -ldl -lpthread -lfst -lfstngram -lfstlookahead -lfstfar" go get github.com/alphacep/vosk-api/go

var model *vosk.VoskModel
var rec *vosk.VoskRecognizer

func InitVoskModel() {
	var err error
	if model != nil {
		return
	}
	log.Printf("VOSK INIT")

	model, err = vosk.NewModel("/opt/vosk-models/vosk-model-small-ru-0.22")
	if err != nil {
		fmt.Printf("Error init model: %s\n", err.Error())

		return
	}
	rec, err = vosk.NewRecognizer(model)
	if err != nil {
		fmt.Printf("Error init recognizer: %s\n", err.Error())

		return
	}
	log.Printf("VOSK DONE")
}

var busy bool = false

func Recognize(filename string) (string, error) {
	var err error

	if model == nil {
		return "", errors.New("vosk not initialized")

	}
	if busy {
		return "", errors.New("busy")
	}
	busy = true

	fileinfo, err := os.Stat(filename)
	if err != nil {
		fmt.Printf("File error: %s\n", err.Error())
		busy = false

		return "", errors.New("file error")
	}
	newFilename := filename + ".waw"
	fileinfo, err = os.Stat(newFilename)
	if err != nil {
		DLog(fmt.Sprintf("New file error (this is not an error): %s\n", err.Error()))
		//ffmpeg -loglevel quiet -i /opt/src/go/.tdlib/files/voice/5467822912058692504.oga -ar 16000 -ac 1 -f s16le 1.waw
		cmd := exec.Command("ffmpeg", "-loglevel", "quiet", "-i", filename, "-ar", "16000", "-ac", "1", "-f", "s16le", newFilename)
		err = cmd.Run()
		if err != nil {
			fmt.Printf("ffmpeg error: %s\n", err.Error())
			busy = false

			return "", errors.New("error exec ffmpeg")
		}
		fileinfo, err = os.Stat(newFilename)
	}

	file, err := os.Open(newFilename)
	if err != nil {
		fmt.Printf("Error loading file: %s\n", err.Error())
		busy = false

		return "", errors.New("error loading file")
	}

	defer file.Close()

	filesize := fileinfo.Size()
	buffer := make([]byte, filesize)

	_, err = file.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading: %s\n", err.Error())
		busy = false

		return "", errors.New("error reading")

	}
	res, err := vosk.VoskFinalResult(rec, buffer), nil
	busy = false

	return res, err
}

func RecognizeByFileId(acc int64, remoteId string) (string, error) {
	file, err := DownloadFileByRemoteId(acc, remoteId)
	if err != nil {

		return "", errors.New("cannot download file: " + err.Error())

	}

	if file.Local.Path == "" {

		return "", errors.New("no local path for file " + remoteId)
	}
	text, err := Recognize(file.Local.Path)

	if err != nil {

		return "", errors.New("cannot recognize: " + err.Error())
	}

	return text, nil
}
