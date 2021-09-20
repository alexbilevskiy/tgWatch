package structs

type TgUpdate struct {
	T    string
	Time int32
	Upd  interface{}
	Raw  []byte
}

type ChatFilter struct {
	Id            int32
	Title         string
	IncludedChats []int64
}

type ChatPosition struct {
	ChatId   int64
	ListId   int32
	Order    int64
	IsPinned bool
}

type ChatCounters struct {
	ChatId   int64
	Counters map[string]int32
}

type IgnoreLists struct {
	T               string
	IgnoreChatIds   map[string]bool
	IgnoreAuthorIds map[string]bool
	IgnoreFolders   map[string]bool
}

type Account struct {
	Id int64
	Phone string
	DbPrefix string
	DataDir string
	Status string
}