package main

import (
	"fmt"

	"github.com/ylacancellera/galera-log-explainer/regex"
)

type regexList struct{}

func (l *regexList) Help() string {
	return "List available regexes"
}

func (l *regexList) Run() error {

	allregexes := regex.AllRegexes()
	keys := []string{}
	for k := range allregexes {
		keys = append(keys, k)

	}
	fmt.Println(keys)
	return nil
}
