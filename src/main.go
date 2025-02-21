package main

import (
	"os"

	"hiku/config"
	"hiku/server"

	"github.com/urfave/cli"
)

func createCliApp() *cli.App {
	app := cli.NewApp()
	app.Usage = "Hiku Scheduler"
	app.UsageText = "hiku COMMAND [ARG...]"
	app.ArgsUsage = "ArgsUsage"
	app.EnableBashCompletion = true
	app.HideVersion = true

	configFlag := cli.StringFlag{
		Name:  "config, c",
		Usage: "Config json file",
		Value: "hiku.json",
	}
	app.Commands = []cli.Command{
		cli.Command{Name: "start", Usage: "Start Hiku",
			UsageText:   "hiku start [-c|--config=FILEPATH]",
			Description: "The scheduler starts with settings from config json file.",
			Flags:       []cli.Flag{configFlag},
			Action: func(c *cli.Context) error {
				cfgFilePath := c.String("config")
				cfg := config.LoadConfigFromFile(cfgFilePath)
				return server.Start(cfg.ToConfig())
			},
		},
	}
	return app
}

func main() {
	app := createCliApp()
	app.Run(os.Args)
}
