package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/cloudfoundry/cli/cf/terminal"
	"github.com/cloudfoundry/cli/cf/trace"
	"github.com/cloudfoundry/cli/plugin"

	"github.com/hpcloud/cf-plugin-backup/cmd"
	"github.com/hpcloud/cf-plugin-backup/commands"
)

var target string

//BackupPlugin represents the struct of the cf cli plugin
type BackupPlugin struct {
	cliConnection plugin.CliConnection
	argLength     int
	ui            terminal.UI
	token         string
}

func main() {
	log.SetOutput(commands.Writer)

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in Backup plugin: %s\n", r)
			fmt.Printf("%s\n", debug.Stack())
		}
	}()

	plugin.Start(new(BackupPlugin))
}

//Run method called before each command
func (c *BackupPlugin) Run(cliConnection plugin.CliConnection, args []string) {
	c.cliConnection = cliConnection
	c.argLength = len(args)

	traceEnv := os.Getenv("CF_TRACE")
	traceLogger := trace.NewLogger(commands.Writer, false, traceEnv, "")

	c.ui = terminal.NewUI(os.Stdin, commands.Writer, terminal.NewTeePrinter(commands.Writer), traceLogger)

	bearer, err := commands.GetBearerToken(cliConnection)
	if err != nil {
		commands.ShowFailed(fmt.Sprint("ERROR:", err))
		return
	}

	c.token = bearer

	if c.argLength == 1 {
		c.showCommandsWithHelpText()
		return
	}

	cmd.CliConnection = cliConnection

	cmd.RootCmd.SetArgs(args[1:])
	cmd.Execute()
}

//GetMetadata returns metadata for cf cli
func (c *BackupPlugin) GetMetadata() plugin.PluginMetadata {
	helpMessages := map[string]string{
		"snapshot": "cf backup snapshot",
		"restore":  "cf backup restore [--include-security-groups]",
		"info":     "cf backup info",
	}
	summary := ""
	for _, value := range helpMessages {
		summary = fmt.Sprintf("%s \n %s ", summary, value)
	}
	return plugin.PluginMetadata{
		Name: "Backup",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "backup",
				HelpText: "View command's help text",
				UsageDetails: plugin.Usage{
					Usage: summary,
				},
			},
			plugin.Command{
				Name:     "backup snapshot",
				HelpText: "Create a backup",
				UsageDetails: plugin.Usage{
					Usage: helpMessages["snapshot"],
				},
			},
			plugin.Command{
				Name:     "backup restore",
				HelpText: "Restore a backup",
				UsageDetails: plugin.Usage{
					Usage: helpMessages["restore"],
				},
			},
			plugin.Command{
				Name:     "backup info",
				HelpText: "Show backup summary",
				UsageDetails: plugin.Usage{
					Usage: helpMessages["info"],
				},
			},
		},
	}
}

func (c *BackupPlugin) showCommandsWithHelpText() {
	metadata := c.GetMetadata()
	for _, command := range metadata.Commands {
		fmt.Printf("%-25s %-50s\n", command.Name, command.HelpText)
	}
	return
}
