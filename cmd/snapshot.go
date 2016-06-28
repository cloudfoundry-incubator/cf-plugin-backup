package cmd

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hpcloud/cf-plugin-backup/models"
	"github.com/hpcloud/cf-plugin-backup/util"
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create a new CloudFoundry backup snapshot",
	Long: `Create a new CloudFoundry backup snapshot to a local file.
`,
	Run: func(cmd *cobra.Command, args []string) {
		backupResources, err := util.GetOrgsResourcesRecurively(&util.CliConnectionCCApi{CliConnection: CliConnection})
		util.FreakOut(err)

		sharedDomains, err := util.GetSharedDomains(&util.CliConnectionCCApi{CliConnection: CliConnection})
		util.FreakOut(err)

		securityGroups, err := util.GetSecurityGroups(&util.CliConnectionCCApi{CliConnection: CliConnection})
		util.FreakOut(err)

		backupJson, err := util.CreateBackupJSON(models.BackupModel{
			Organizations:  backupResources,
			SharedDomains:  sharedDomains,
			SecurityGroups: securityGroups,
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

		err = os.Mkdir(filepath.Join(BackupDir, BackupAppBitsDir), 0755)
		if err != nil && !os.IsExist(err) {
			util.FreakOut(err)
		}

		resources := util.RestoreOrgResourceModels(backupModel.Organizations)
		for _, org := range *resources {
			for _, space := range *(org.Entity["spaces"].(*[]*models.ResourceModel)) {
				for _, app := range *(space.Entity["apps"].(*[]*models.ResourceModel)) {
					appGuid := app.Metadata["guid"].(string)
					log.Println("Saving bits for application", app.Entity["name"], appGuid)

					appZipPath := filepath.Join(BackupDir, BackupAppBitsDir, appGuid+".zip")
					err := appBits.SaveDroplet(appGuid, appZipPath)
					if err != nil {
						log.Printf("Could not save bits for %v: %v", appGuid, err)
					}
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
