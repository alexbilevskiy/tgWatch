package config

type ConfigFileStruct struct {
	ApiId   int32  `json:"ApiId"`
	ApiHash string `json:"ApiHash"`
	WebListen string `json:"WebListen"`
	Mongo map[string]string `json:"Mongo"`
	Debug bool `json:"Debug"`
}
