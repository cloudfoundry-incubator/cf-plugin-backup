package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/cloudfoundry/cli/plugin"
	"github.com/spf13/cobra"

	"github.com/hpcloud/cf-plugin-backup/models"
	"github.com/hpcloud/cf-plugin-backup/util"
)

func GetAllOrganizations(cliConnection plugin.CliConnection) string {
	resources := util.GetResources(cliConnection, "/v2/organizations", 1, nil, nil)
	jsonResources, err := json.MarshalIndent(resources, "", " ")
	util.FreakOut(err)
	return string(jsonResources)
}

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create a new CloudFoundry backup snapshot",
	Long: `Create a new CloudFoundry backup snapshot to a local file.
`,
	Run: func(cmd *cobra.Command, args []string) {
		backupResources, err := util.GetOrgsResourcesRecurively(CliConnection)
		util.FreakOut(err)

		backupJson, err := util.CreateBackupJson(models.BackupModel{Organizations: backupResources})
		util.FreakOut(err)

		fmt.Println(backupJson)

		err = ioutil.WriteFile(BackupFile, []byte(backupJson), 0644)
		util.FreakOut(err)
	},
}

func init() {
	RootCmd.AddCommand(snapshotCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// snapshotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// snapshotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
