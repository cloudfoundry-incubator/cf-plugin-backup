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

const (
	ORG_AUDIT     = "audited_organizations"
	ORG_BILLING   = "billing_managed_organizations"
	ORG_MANAGER   = "managed_organizations"
	ORG_DEV       = "organizations"
	SPACE_AUDIT   = "audited_spaces"
	SPACE_MANAGER = "managed_spaces"
	SPACE_DEV     = "spaces"
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

func restoreUser(user, space, role string) string {
	showInfo(fmt.Sprintf("Restaurating User: %s", user))

	user_id := getUserId(user)
	if user_id == "" {
		showWarning(fmt.Sprintf("Could not find user: %s", user))
	} else {
		path := fmt.Sprintf("/v2/users/%s/%s/%s", user_id, role, space)
		resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
			path, "-X", "PUT")
		if err != nil {
			showWarning(fmt.Sprintf("Could not create user association %s, exception message: %s",
				user, err.Error()))
		}
		return showResult(resp, "user", user, false)
	}
	return ""
}

func getUserId(user string) string {
	cache := make(map[string]interface{})
	resources := util.GetResources(CliConnection, "/v2/users", 1, nil, cache)
	for _, u := range *resources {
		if u.Entity["username"].(string) == user {
			return u.Metadata["guid"].(string)
		}
	}

	return ""
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
		inName := fmt.Sprintf("%v", oResp["entity"].(map[string]interface{})["name"])
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

func showResult(resp []string, entity, name string, checkName bool) string {
	oResp := make(map[string]interface{})
	if len(resp) == 0 {
		showWarning(fmt.Sprintf("Got null response while restoring %s %s",
			entity, name))
		return ""
	}
	err := json.Unmarshal([]byte(resp[0]), &oResp)
	if err != nil {
		showWarning(fmt.Sprintf("Got unknow response while restoring %s %s: %s",
			entity, name, err.Error()))
		return ""
	}
	if oResp["error_code"] != nil {
		showWarning(fmt.Sprintf("got %v-%v while restoring %s %s",
			oResp["error_code"], oResp["description"], entity, name))
		return ""
	}

	if checkName {
		if oResp["entity"] != nil {

			resource := util.TransformToResource(oResp, make(map[string]interface{}), nil)
			inName := fmt.Sprintf("%v", resource.Entity["name"])
			if inName == name {
				showInfo(fmt.Sprintf("Succesfully restored %s %s", entity, name))
				if resource.Metadata != nil {
					return resource.Metadata["guid"].(string)
				}
			} else {
				showWarning(fmt.Sprintf("Name %s does not match requested name %s",
					oResp["name"], name))
				return ""
			}
		} else {
			showWarning(fmt.Sprintln("\tWarning unknown answer received"))
			return ""
		}
	}

	return ""
}

func restoreFromJSON() {
	var fileContent []byte
	_, err := os.Stat(BackupFile)
	util.FreakOut(err)

	fileContent, err = ioutil.ReadFile(BackupFile)
	util.FreakOut(err)

	backupObject, err := util.ReadBackupJSON(fileContent)
	util.FreakOut(err)

	orgs := util.RestoreOrgResourceModels(backupObject.Organizations)
	for _, org := range *orgs {
		o := Org{Name: org.Entity["name"].(string), QuotaGUID: org.Entity["quota_definition_guid"].(string)}
		org_guid := restoreOrg(o)

		if org_guid != "" {
			auditors := org.Entity["auditors"].(*[]*models.ResourceModel)
			for _, auditor := range *auditors {
				restoreUser(auditor.Entity["username"].(string), org_guid, ORG_DEV)
				restoreUser(auditor.Entity["username"].(string), org_guid, ORG_AUDIT)
			}

			billing_managers := org.Entity["billing_managers"].(*[]*models.ResourceModel)
			for _, manager := range *billing_managers {
				restoreUser(manager.Entity["username"].(string), org_guid, ORG_DEV)
				restoreUser(manager.Entity["username"].(string), org_guid, ORG_BILLING)
			}

			managers := org.Entity["managers"].(*[]*models.ResourceModel)
			for _, manager := range *managers {
				restoreUser(manager.Entity["username"].(string), org_guid, ORG_DEV)
				restoreUser(manager.Entity["username"].(string), org_guid, ORG_MANAGER)
			}

			spaces := org.Entity["spaces"].(*[]*models.ResourceModel)
			for _, space := range *spaces {
				s := Space{Name: space.Entity["name"].(string), OrganizationGuid: org_guid}
				space_guid := restoreSpace(s)

				if space_guid != "" {
					auditors := space.Entity["auditors"].(*[]*models.ResourceModel)
					for _, auditor := range *auditors {
						restoreUser(auditor.Entity["username"].(string), space_guid, SPACE_AUDIT)
					}

					developers := space.Entity["developers"].(*[]*models.ResourceModel)
					for _, developer := range *developers {
						restoreUser(developer.Entity["username"].(string), space_guid, SPACE_DEV)
					}

					managers := space.Entity["managers"].(*[]*models.ResourceModel)
					for _, manager := range *managers {
						restoreUser(manager.Entity["username"].(string), space_guid, SPACE_MANAGER)
					}
				}
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
