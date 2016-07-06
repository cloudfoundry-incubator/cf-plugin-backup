package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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

type App struct {
	Name               string      `json:"name"`
	SpaceGuid          string      `json:"space_guid"`
	Diego              interface{} `json:"diego"`
	Ports              interface{} `json:"ports"`
	Memory             interface{} `json:"memory"`
	Instances          interface{} `json:"instances"`
	DiskQuota          interface{} `json:"disk_quota"`
	StackGuid          string      `json:"stack_guid"`
	Command            interface{} `json:"command"`
	Buildpack          interface{} `json:"buildpack"`
	HealthCheckType    interface{} `json:"health_check_type"`
	HealthCheckTimeout interface{} `json:"health_check_timeout"`
	EnableSSH          interface{} `json:"enable_ssh"`
	DockerImage        interface{} `json:"docker_image"`
	EnvironmentJson    interface{} `json:"environment_json"`
	State              interface{} `json:"state"`
}

type Route struct {
	DomainGuid string      `json:"domain_guid"`
	SpaceGuid  string      `json:"space_guid"`
	Port       interface{} `json:"port"`
	Host       interface{} `json:"host"`
	Path       interface{} `json:"path"`
}

func showInfo(sMessage string) {
	fmt.Println(sMessage)
}

func showWarning(sMessage string) {
	fmt.Printf("WARNING: %s\n", sMessage)
}

func restoreUserRole(user, space, role string) {
	showInfo(fmt.Sprintf("Restoring role for User: %s", user))

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
		showResult(resp, "user", "", "", false)
	}
}

func getUserId(user string) string {
	resources := util.GetResources(CliConnection, "/v2/users", 1)
	for _, u := range *resources {
		if u.Entity["username"].(string) == user {
			return u.Metadata["guid"].(string)
		}
	}

	return ""
}

func restoreOrg(org Org) string {
	showInfo(fmt.Sprintf("Restoring Organization: %s", org.Name))
	oJson, err := json.Marshal(org)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/organizations", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create org %s, exception message: %s",
			org.Name, err.Error()))
	}
	return showResult(resp, "org", "name", org.Name, true)
}

func restoreApp(app App) string {
	showInfo(fmt.Sprintf("Restoring App: %s", app.Name))
	oJson, err := json.Marshal(app)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/apps", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create app %s, exception message: %s",
			app.Name, err.Error()))
	}
	return showResult(resp, "app", "name", app.Name, true)
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
	return showResult(resp, "space", "name", space.Name, true)
}

func showResult(resp []string, entity, checkField, expectedValue string, check bool) string {
	oResp := make(map[string]interface{})
	if len(resp) == 0 {
		showWarning(fmt.Sprintf("Got null response while restoring %s %s",
			entity, expectedValue))
		return ""
	}
	err := json.Unmarshal([]byte(util.ConcatStringArray(resp)), &oResp)
	if err != nil {
		showWarning(fmt.Sprintf("Got unknown response while restoring %s %s: %s",
			entity, expectedValue, err.Error()))
		return ""
	}
	if oResp["error_code"] != nil {
		showWarning(fmt.Sprintf("got %v-%v while restoring %s %s",
			oResp["error_code"], oResp["description"], entity, expectedValue))
		return ""
	}

	if check {
		if oResp["entity"] != nil {
			inName := (oResp["entity"].(map[string]interface{}))[checkField].(string)
			if inName == expectedValue {
				showInfo(fmt.Sprintf("Succesfully restored %s %s", entity, expectedValue))
				if oResp["metadata"] != nil {
					return (oResp["metadata"].(map[string]interface{}))["guid"].(string)
				}
			} else {
				showWarning(fmt.Sprintf("Field %s does not match requested value %s",
					oResp[checkField], expectedValue))
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
				restoreUserRole(auditor.Entity["username"].(string), org_guid, ORG_DEV)
				restoreUserRole(auditor.Entity["username"].(string), org_guid, ORG_AUDIT)
			}

			billing_managers := org.Entity["billing_managers"].(*[]*models.ResourceModel)
			for _, manager := range *billing_managers {
				restoreUserRole(manager.Entity["username"].(string), org_guid, ORG_DEV)
				restoreUserRole(manager.Entity["username"].(string), org_guid, ORG_BILLING)
			}

			managers := org.Entity["managers"].(*[]*models.ResourceModel)
			for _, manager := range *managers {
				restoreUserRole(manager.Entity["username"].(string), org_guid, ORG_DEV)
				restoreUserRole(manager.Entity["username"].(string), org_guid, ORG_MANAGER)
			}

			spaces := org.Entity["spaces"].(*[]*models.ResourceModel)
			for _, space := range *spaces {
				s := Space{Name: space.Entity["name"].(string), OrganizationGuid: org_guid}
				space_guid := restoreSpace(s)

				if space_guid != "" {
					auditors := space.Entity["auditors"].(*[]*models.ResourceModel)
					for _, auditor := range *auditors {
						restoreUserRole(auditor.Entity["username"].(string), space_guid, SPACE_AUDIT)
					}

					developers := space.Entity["developers"].(*[]*models.ResourceModel)
					for _, developer := range *developers {
						restoreUserRole(developer.Entity["username"].(string), space_guid, SPACE_DEV)
					}

					managers := space.Entity["managers"].(*[]*models.ResourceModel)
					for _, manager := range *managers {
						restoreUserRole(manager.Entity["username"].(string), space_guid, SPACE_MANAGER)
					}
				}

				apps := space.Entity["apps"].(*[]*models.ResourceModel)
				packager := &util.CFPackager{
					Cli:    CliConnection,
					Writer: new(util.CFFileWriter),
					Reader: new(util.CFFileReader),
				}
				appBits := util.NewCFDroplet(CliConnection, packager)

				for _, app := range *apps {
					stackName := app.Entity["stack"].(*models.ResourceModel).Entity["name"].(string)
					stackGuid := getStackGuid(stackName)
					if stackGuid == "" {
						showWarning(fmt.Sprintln("Stack %s not found. Skipping app %s", stackName, app.Entity["name"].(string)))
						continue
					}

					a := App{
						Name:               app.Entity["name"].(string),
						SpaceGuid:          space_guid,
						Diego:              app.Entity["diego"],
						Memory:             app.Entity["memory"],
						Instances:          app.Entity["instances"],
						DiskQuota:          app.Entity["disk_quota"],
						StackGuid:          stackGuid,
						Command:            app.Entity["command"],
						Buildpack:          app.Entity["buildpack"],
						HealthCheckType:    app.Entity["health_check_type"],
						HealthCheckTimeout: app.Entity["health_check_timeout"],
						EnableSSH:          app.Entity["enable_ssh"],
						DockerImage:        app.Entity["docker_image"],
						EnvironmentJson:    app.Entity["environment_json"],
						Ports:              app.Entity["ports"],
					}

					appGuid := restoreApp(a)

					if dockerImg, hit := app.Entity["docker_image"]; !hit || dockerImg == nil {
						oldAppGuid := app.Metadata["guid"].(string)
						appZipPath := filepath.Join(BackupDir, BackupAppBitsDir, oldAppGuid+".zip")
						err = appBits.UploadDroplet(appGuid, appZipPath)
						if err != nil {
							showWarning(fmt.Sprintf("Could not upload app bits for app %s: %s", app.Entity["name"].(string), err.Error()))
						}
					}

					state := app.Entity["state"].(string)
					a.State = state
					updateApp(appGuid, a)

					routes := app.Entity["routes"].(*[]*models.ResourceModel)
					for _, route := range *routes {
						domain := route.Entity["domain"].(*models.ResourceModel)
						domainName := domain.Entity["name"].(string)
						domainGuid := getSharedDomainGuid(domainName)
						if domainGuid == "" {
							showWarning(fmt.Sprintf("Could not find shared domain %s", domainName))
							continue
						}

						r := Route{
							DomainGuid: domainGuid,
							SpaceGuid:  space_guid,
							Port:       route.Entity["port"],
							Path:       route.Entity["path"],
							Host:       route.Entity["host"],
						}

						routeGuid := createRoute(r)
						bindRoute(appGuid, routeGuid)
					}
				}
			}
		}
	}
}

func bindRoute(appGuid, routeGuid string) string {
	showInfo(fmt.Sprintf("Binding route %s to app %s", routeGuid, appGuid))

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/apps/"+appGuid+"/routes/"+routeGuid, "-H", "Content-Type: application/x-www-form-urlencoded",
		"-X", "PUT")
	if err != nil {
		showWarning(fmt.Sprintf("Could not bind route %s, exception message: %s",
			routeGuid, err.Error()))
	}
	return showResult(resp, "route binding", "", "", false)
}

func createRoute(route Route) string {
	showInfo(fmt.Sprintf("Creating route: %s", route.Host))
	oJson, err := json.Marshal(route)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/routes", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create route %s, exception message: %s",
			route.Host, err.Error()))
	}
	return showResult(resp, "route", "host", route.Host.(string), true)
}

func getSharedDomainGuid(domainName string) string {
	resources := util.GetResources(CliConnection, "/v2/shared_domains?q=name:"+domainName, 1)
	for _, u := range *resources {
		if u.Entity["name"].(string) == domainName {
			return u.Metadata["guid"].(string)
		}
	}

	return ""
}

func getStackGuid(stackName string) string {
	resources := util.GetResources(CliConnection, "/v2/stacks?q=name:"+stackName, 1)
	for _, u := range *resources {
		if u.Entity["name"].(string) == stackName {
			return u.Metadata["guid"].(string)
		}
	}

	return ""
}

func updateApp(guid string, app App) {
	showInfo(fmt.Sprintf("Updating app %s", app.Name))
	oJson, err := json.Marshal(app)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/apps/"+guid, "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "PUT")
	if err != nil {
		showWarning(fmt.Sprintf("Could not update app %s, exception message: %s",
			app.Name, err.Error()))
	}
	showResult(resp, "app", "name", app.Name, true)
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
