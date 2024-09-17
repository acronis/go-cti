package _package

type LangCode string

type Field string

type Entry map[Field]string

type Dictionary map[LangCode]Entry

type Dictionaries struct {
	Dictionaries Dictionary `json:"dictionaries"`
}
