package main

import (
	"errors"
	"fmt"

	"github.com/DTFN/dtfn/cmd/utils"
	"github.com/tendermint/tendermint/libs/common"
	"gopkg.in/urfave/cli.v1"
)

type simpleLogger struct{}

func (*simpleLogger) Info(msg string, keyvals ...interface{}) {
	fmt.Println(msg)
	fmt.Println(keyvals...)
}

func standaloneCmd(ctx *cli.Context) error {
	node, backend := utils.MakeStandaloneNode(ctx)
	if node == nil || backend == nil {
		return errors.New("cannot start node or backend")
	}

	startNode(ctx, node)

	logger := &simpleLogger{}
	common.TrapSignal(logger, func() {

	})

	select {}

	return nil
}
