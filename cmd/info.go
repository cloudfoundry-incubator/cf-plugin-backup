package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/hpcloud/cf-plugin-backup/models"
	"github.com/hpcloud/cf-plugin-backup/util"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show information about the current snapshot",
	Long: `Show information about the current snapshot.
It includes a summary of organizations, spaces and apps
	`,
	Run: func(cmd *cobra.Command, args []string) {
		backupJson, err := ioutil.ReadFile(BackupFile)
		util.FreakOut(err)

		backupModel := models.BackupModel{}
		err = json.Unmarshal(backupJson, &backupModel)
		util.FreakOut(err)

		resources := util.TransformToResources(backupModel.Organizations, make(map[string]interface{}))
		for _, org := range *resources {
			fmt.Println("-", "Org ", org.Entity["name"])
			for _, space := range *(org.Entity["spaces"].(*[]*models.ResourceModel)) {
				fmt.Println("--", "Space ", space.Entity["name"])
				for _, app := range *(space.Entity["apps"].(*[]*models.ResourceModel)) {
					fmt.Println("---", "App ", app.Entity["name"])
				}
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(infoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// infoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// infoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
