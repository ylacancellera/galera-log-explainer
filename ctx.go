package main

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type ctx struct {
	list
}

func (c *ctx) Help() string {
	return "Dump the context derived from the log"
}

func (c *ctx) Run() error {

	if len(c.Paths) != 1 {
		return errors.New("Can only use 1 path at a time for ctx subcommand")
	}

	// for this use case, why restrict regexes
	c.list.All = true
	timeline, err := timelineFromPaths(c.Paths, c.list.regexesToUse(), CLI.Since, CLI.Until)
	if err != nil {
		return err
	}

	for _, t := range timeline {
		out, err := json.MarshalIndent(t[len(t)-1].Ctx, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
	}

	return nil
}
