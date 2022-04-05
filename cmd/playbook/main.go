package main

import (
	"context"
	"flag"
	"github.com/superisaac/rpcmap/cmd/cmdutil"
	"github.com/superisaac/rpcmap/playbook"
	"os"
)

func StartPlaybook() {
	flagset := flag.NewFlagSet("rpcmap-playbook", flag.ExitOnError)

	// connect to server
	pConnect := flagset.String("c", "h2c://127.0.0.1:6000", "connect to rpcmap server")

	// logging flags
	pYaml := flagset.String("config", "playbook.yml", "path to playbook.yml")

	// logging flags
	pLogfile := flagset.String("log", "", "path to log output, default is stdout")

	// parse command-line flags
	flagset.Parse(os.Args[1:])
	cmdutil.SetupLogger(*pLogfile)

	pb := playbook.NewPlaybook()
	if err := pb.Config.Load(*pYaml); err != nil {
		panic(err)
	}

	if err := pb.Run(context.Background(), *pConnect); err != nil {
		panic(err)
	}
}

func main() {
	StartPlaybook()
}
