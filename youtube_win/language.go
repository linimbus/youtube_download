package main

import (
	"github.com/astaxie/beego/logs"
	"gopkg.in/yaml.v2"
)

type LangItem struct {
	Key     string 	`json:"key"`
	Value []string  `json:"value"`
}

type LangCtrl struct {
	idx    int
	cache  map[string]*LangItem
	Items  []LangItem
}

var langCtrl *LangCtrl

func LanguageInit() error {
	langCtrl = new(LangCtrl)
	langCtrl.Items = make([]LangItem, 0)
	langCtrl.cache = make(map[string]*LangItem, 1024)

	body, err := BoxFile().Bytes("language.yaml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(body, &langCtrl.Items)
	if err != nil {
		return err
	}

	for _, v := range langCtrl.Items {
		langCtrl.cache[v.Key] = &LangItem{
			Key: v.Key, Value: v.Value,
		}
	}

	length := len(LangOptionGet())
	idx := int(DataIntValueGet("Language"))
	if length <= idx {
		LangOptionSet(0)
	} else {
		langCtrl.idx = idx
	}
	return nil
}

func LangOptionIdx() int {
	return int(DataIntValueGet("Language"))
}

func LangOptionGet() []string {
	item, _ := langCtrl.cache["language"]
	if item == nil {
		return []string{}
	}
	return item.Value
}

func LangOptionSet(idx int)  {
	langCtrl.idx = idx
	DataIntValueSet("language", uint32(idx))
}

func LangValue(key string) string {
	item, _ := langCtrl.cache[key]
	if item == nil {
		logs.Error("lang value %s fail", key)
		return ""
	}
	return item.Value[langCtrl.idx]
}

