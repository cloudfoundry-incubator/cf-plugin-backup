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
	"github.com/hpcloud/termui/termprogressbar"
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create a new CloudFoundry backup snapshot",
	Long: `Create a new CloudFoundry backup snapshot to a local file.
`,
	Run: func(cmd *cobra.Command, args []string) {
		var currentIndex int

		backupResources, err := util.GetOrgsResourcesRecurively(&util.CliConnectionCCApi{CliConnection: CliConnection})
		util.FreakOut(err)
		log.Println("orgs done")
		sharedDomains, err := util.GetSharedDomains(&util.CliConnectionCCApi{CliConnection: CliConnection})
		util.FreakOut(err)
		log.Println("shared domains done")
		securityGroups, err := util.GetSecurityGroups(&util.CliConnectionCCApi{CliConnection: CliConnection})
		util.FreakOut(err)
		log.Println("groups done")
		backupJson, err := util.CreateBackupJSON(models.BackupModel{
			Organizations:  backupResources,
			SharedDomains:  sharedDomains,
			SecurityGroups: securityGroups,
		})
		util.FreakOut(err)

		err = ioutil.WriteFile(BackupFile, []byte(backupJson), 0644)
		util.FreakOut(err)

		// Save app bits

		packager := &util.CFPackager{
			Cli:    CliConnection,
			Writer: new(util.CFFileWriter),
			Reader: new(util.CFFileReader),
		}
		appBits := util.NewCFDroplet(CliConnection, packager)

		backupModel := models.BackupModel{}
		err = json.Unmarshal([]byte(backupJson), &backupModel)
		util.FreakOut(err)

		err = os.Mkdir(filepath.Join(BackupDir, BackupAppBitsDir), 0755)
		if err != nil && !os.IsExist(err) {
			util.FreakOut(err)
		}

		var appsToBackup []*models.ResourceModel

		resources := util.RestoreOrgResourceModels(backupModel.Organizations)
		for _, org := range *resources {
			for _, space := range *(org.Entity["spaces"].(*[]*models.ResourceModel)) {
				for _, app := range *(space.Entity["apps"].(*[]*models.ResourceModel)) {
					if dockerImg, hit := app.Entity["docker_image"]; !hit || dockerImg == nil {
						appsToBackup = append(appsToBackup, app)
					}
				}
			}
		}

		log.Printf("Saving bits for %d apps", len(appsToBackup))

		termuiPGBar := termuiprogressbar.NewProgressBar(len(appsToBackup), true)
		termuiPGBar.Start()

		for _, app := range appsToBackup {
			appGuid := app.Metadata["guid"].(string)
			if currentIndex < len(appsToBackup) {
				termuiPGBar.Increment()
				currentIndex++
			}
			appZipPath := filepath.Join(BackupDir, BackupAppBitsDir, appGuid+".zip")
			err := appBits.SaveDroplet(appGuid, appZipPath)
			if err != nil {
				log.Printf("Could not save bits for %v: %v", appGuid, err)
			}
		}
		termuiPGBar.FinishPrint("App bits saved")
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
