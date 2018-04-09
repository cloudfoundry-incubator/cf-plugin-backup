package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/blang/semver"

	"github.com/SUSE/termui"
	"github.com/SUSE/termui/termpassword"

	"code.cloudfoundry.org/cli/plugin"

	"github.com/SUSE/cf-plugin-backup/cmd"
	"github.com/SUSE/cf-plugin-backup/commands"
	"github.com/SUSE/cf-plugin-backup/util"
)

var target string
var version string

//BackupPlugin represents the struct of the cf cli plugin
type BackupPlugin struct {
	cliConnection plugin.CliConnection
	argLength     int
	ui            *termui.UI
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

	c.ui = termui.New(os.Stdin, commands.Writer, termpassword.NewReader())

	if c.argLength == 1 {
		switch args[0] {
		case "CLI-MESSAGE-UNINSTALL":
			return
		}
	}

	bearer, err := commands.GetBearerToken(cliConnection)
	if err != nil {
		commands.ShowFailed(fmt.Sprint("ERROR:", err))
		return
	}

	c.token = bearer

	isAdmin, err := util.CheckUserScope(bearer, "cloud_controller.admin")
	if err != nil {
		commands.ShowFailed(fmt.Sprint("ERROR:", err))
		return
	}

	if !isAdmin {
		commands.ShowFailed("ERROR: Logged in user has no admin scope.")
		return
	}

	cmd.CliConnection = cliConnection

	cmd.RootCmd.SetArgs(args)
	cmd.Execute()
}

//GetMetadata returns metadata for cf cli
func (c *BackupPlugin) GetMetadata() plugin.PluginMetadata {
	helpMessages := map[string]string{
		"snapshot": "cf backup snapshot",
		"restore":  "cf backup restore [--include-security-groups] [--include-quota-definitions]",
		"info":     "cf backup info",
	}
	summary := ""
	for _, value := range helpMessages {
		summary = fmt.Sprintf("%s \n %s ", summary, value)
	}
	pluginVersion, err := semver.ParseTolerant(version)

	if err != nil {
		panic(fmt.Sprintf("Invalid plugin version %s: %s", version, err))
	}

	return plugin.PluginMetadata{
		Name: "backup",
		Version: plugin.VersionType{
			Major: int(pluginVersion.Major),
			Minor: int(pluginVersion.Minor),
			Build: int(pluginVersion.Patch),
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 14,
			Build: 0,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     "backup-snapshot",
				HelpText: "Create a new CloudFoundry backup snapshot to a local file",
				UsageDetails: plugin.Usage{
					Usage: helpMessages["snapshot"],
				},
			},
			plugin.Command{
				Name:     "backup-restore",
				HelpText: "Restore the CloudFoundry state from a backup created with the snapshot command",
				UsageDetails: plugin.Usage{
					Usage: helpMessages["restore"],
				},
			},
			plugin.Command{
				Name:     "backup-info",
				HelpText: "Show information about the current snapshot",
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
