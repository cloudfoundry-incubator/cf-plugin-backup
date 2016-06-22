package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hpcloud/cf-plugin-backup/models"
	"github.com/hpcloud/cf-plugin-backup/util"

	"github.com/spf13/cobra"
)

type Org struct {
	Name      string `json:"name"`
	QuotaGUID string `json:-`
}

type Space struct {
	Name             string `json:"name"`
	OrganizationGuid string `json:"organization_guid"`
}

func showInfo(sMessage string) {
	fmt.Println(sMessage)
}

func showWarning(sMessage string) {
	fmt.Printf("WARNING: %s\n", sMessage)
}

func restoreOrg(org Org) string {
	showInfo(fmt.Sprintf("Restaurating Org: %s", org.Name))
	oJson, err := json.Marshal(org)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/organizations", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create org %s, exception message: %s",
			org.Name, err.Error()))
	}
	return showOrgResult(resp, org)
}

func restoreSpace(space Space) string {
	showInfo(fmt.Sprintf("Restaurating Space: %s", space.Name))
	oJson, err := json.Marshal(space)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/spaces", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create space %s, exception message: %s",
			space.Name, err.Error()))
	}
	return showSpaceResult(resp, space)
}

func showSpaceResult(resp []string, space Space) string {
	oResp := make(map[string]interface{})
	if len(resp) == 0 {
		showWarning(fmt.Sprintf("Got null response while restoring space %s",
			space.Name))
		return ""
	}
	err := json.Unmarshal([]byte(resp[0]), &oResp)
	if err != nil {
		showWarning(fmt.Sprintf("Got unknow response while restoring space %s: %s",
			space.Name, err.Error()))
		return ""
	}
	if oResp["error_code"] != nil {
		showWarning(fmt.Sprintf("got %v-%v while restoring space %s",
			oResp["error_code"], oResp["description"], space.Name))
		return ""
	}

	if oResp["entity"] != nil {

		resource := util.TransformToResource(oResp, make(map[string]interface{}), nil)

		inName := fmt.Sprintf("%v", resource.Entity["name"])
		if inName == space.Name {
			showInfo(fmt.Sprintf("Succesfully restored space %s", space.Name))
		} else {
			showWarning(fmt.Sprintf("Name %s does not match reqyested name %s",
				oResp["name"], space.Name))
		}

		return resource.Metadata["guid"].(string)
	} else {
		showWarning(fmt.Sprintln("\tWarning unknown answer received"))
		return ""
	}
}

func showOrgResult(resp []string, org Org) string {
	oResp := make(map[string]interface{})
	if len(resp) == 0 {
		showWarning(fmt.Sprintf("Got null response while restoring org %s",
			org.Name))
		return ""
	}
	err := json.Unmarshal([]byte(resp[0]), &oResp)
	if err != nil {
		showWarning(fmt.Sprintf("Got unknow response while restoring org %s: %s",
			org.Name, err.Error()))
		return ""
	}
	if oResp["error_code"] != nil {
		showWarning(fmt.Sprintf("got %v-%v while restoring org %s",
			oResp["error_code"], oResp["description"], org.Name))
		return ""
	}

	if oResp["entity"] != nil {

		resource := util.TransformToResource(oResp, make(map[string]interface{}), nil)

		inName := fmt.Sprintf("%v", resource.Entity["name"])
		if inName == org.Name {
			showInfo(fmt.Sprintf("Succesfully restored org %s", org.Name))
		} else {
			showWarning(fmt.Sprintf("Name %s does not match reqyested name %s",
				oResp["name"], org.Name))
		}

		return resource.Metadata["guid"].(string)
	} else {
		showWarning(fmt.Sprintln("\tWarning unknown answer received"))
		return ""
	}
}

func restoreFromJSON() {
	var fileContent []byte
	_, err := os.Stat(BackupFile)
	util.FreakOut(err)

	fileContent, err = ioutil.ReadFile(BackupFile)
	util.FreakOut(err)

	backupObject, err := util.ReadBackupJSON(fileContent)
	util.FreakOut(err)

	orgs := util.TransformToResources(backupObject.Organizations, make(map[string]interface{}), nil)
	for _, org := range *orgs {
		o := Org{Name: org.Entity["name"].(string), QuotaGUID: org.Entity["quota_definition_guid"].(string)}
		org_guid := restoreOrg(o)

		if org_guid != "" {
			spaces := org.Entity["spaces"].(*[]*models.ResourceModel)
			for _, space := range *spaces {
				s := Space{Name: space.Entity["name"].(string), OrganizationGuid: org_guid}
				restoreSpace(s)
			}
		}
	}
}

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore the CloudFoundry state from a backup",
	Long: `Restore the CloudFoundry state from a backup created with the snapshot command
`,
	Run: func(cmd *cobra.Command, args []string) {
		restoreFromJSON()
	},
}

func init() {
	RootCmd.AddCommand(restoreCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// restoreCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// restoreCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
