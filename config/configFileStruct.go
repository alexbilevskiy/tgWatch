package config

type ConfigFileStruct struct {
	ApiId   int32  `json:"ApiId"`
	ApiHash string `json:"ApiHash"`
	IgnoreChatIds map[string]string `json:"IgnoreChatIds"`
	IgnoreAuthorIds map[string]string `json:"IgnoreAuthorIds"`
	IgnoreFolders map[string]bool `json:"IgnoreFolders"`
	WebListen string `json:"WebListen"`
	Mongo map[string]string `json:"Mongo"`
}
