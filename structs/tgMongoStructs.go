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
	Order    int64
	IsPinned bool
}

type ChatCounters struct {
	ChatId   int64
	Counters map[string]int32
}
