package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

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

		sharedDomains, err := util.GetSharedDomains(CliConnection)
		util.FreakOut(err)

		backupJson, err := util.CreateBackupJSON(models.BackupModel{
			Organizations: backupResources,
			SharedDomains: sharedDomains,
		})
		util.FreakOut(err)

		err = ioutil.WriteFile(BackupFile, []byte(backupJson), 0644)
		util.FreakOut(err)

		// Save app bits

		downloader := &util.CFDownloader{
			Cli:    CliConnection,
			Writer: new(util.CFFileWriter),
		}
		appBits := util.NewCFDroplet(CliConnection, downloader)

		backupModel := models.BackupModel{}
		err = json.Unmarshal([]byte(backupJson), &backupModel)
		util.FreakOut(err)

		resources := util.TransformToResources(backupModel.Organizations, make(map[string]interface{}))
		for _, org := range *resources {
			for _, space := range *(org.Entity["spaces"].(*[]*models.ResourceModel)) {
				for _, app := range *(space.Entity["apps"].(*[]*models.ResourceModel)) {
					appGuid := app.Metadata["guid"].(string)
					fmt.Println("Saving bits its for application", app.Entity["name"], appGuid)

					appZipPath := filepath.Join(BackupDir, BackupAppBitsDir, appGuid+".zip")
					err := appBits.SaveDroplet(appGuid, appZipPath)
					util.FreakOut(err)
				}
			}
		}
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
