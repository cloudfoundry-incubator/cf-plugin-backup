package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

type SecurityGroup struct {
	Name       string      `json:"name"`
	Rules      interface{} `json:"rules"`
	SpaceGuids []string    `json:"space_guids"`
}

type SharedDomain struct {
	Name string `json:"name"`
}

type PrivateDomain struct {
	Name                   string `json:"name"`
	OwningOrganizationGuid string `json:"owning_organization_guid"`
}

func showInfo(sMessage string) {
	log.Printf(sMessage)
}

func showWarning(sMessage string) {
	log.Printf("WARNING: %s\n", sMessage)
}

func restorePrivateDomain(domain PrivateDomain) (string, error) {
	showInfo(fmt.Sprintf("Restoring private domain: %s", domain.Name))
	oJson, err := json.Marshal(domain)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/private_domains", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create private domain %s, exception message: %s",
			domain.Name, err.Error()))
	}
	result, err := getResult(resp, "name", domain.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error restoring private domain %s: %s", domain.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully restored private domain %s", domain.Name))
	}
	return result, nil
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
		_, err = getResult(resp, "", "")
		if err != nil {
			showWarning(fmt.Sprintf("Error restoring user role %s for user %s: %s", role, user, err.Error()))
		} else {
			showInfo(fmt.Sprintf("Succesfully restored user role %s for user %s", role, user))
		}
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
	showInfo(fmt.Sprintf("Restoring organization: %s", org.Name))
	oJson, err := json.Marshal(org)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/organizations", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create organization %s, exception message: %s",
			org.Name, err.Error()))
	}
	result, err := getResult(resp, "name", org.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error restoring organization %s: %s", org.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully restored organization %s", org.Name))
	}
	return result
}

func restoreApp(app App) string {
	oJson, err := json.Marshal(app)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/apps", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create application %s, exception message: %s",
			app.Name, err.Error()))
	}
	result, err := getResult(resp, "name", app.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error restoring application %s: %s", app.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully restored application %s", app.Name))
	}
	return result
}

func restoreSpace(space Space) string {
	showInfo(fmt.Sprintf("Restoring space: %s", space.Name))
	oJson, err := json.Marshal(space)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/spaces", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create space %s, exception message: %s",
			space.Name, err.Error()))
	}
	result, err := getResult(resp, "name", space.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error restoring space %s: %s", space.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully restored space %s", space.Name))
	}
	return result
}

func getResult(resp []string, checkField, expectedValue string) (string, error) {
	oResp := make(map[string]interface{})
	if len(resp) == 0 {
		return "", fmt.Errorf("Got null response")
	}
	err := json.Unmarshal([]byte(util.ConcatStringArray(resp)), &oResp)
	if err != nil {
		return "", err
	}
	if oResp["error_code"] != nil {
		return "", fmt.Errorf("Got %v-%v", oResp["error_code"], oResp["description"])
	}

	if checkField != "" {
		if oResp["entity"] != nil {
			inName := (oResp["entity"].(map[string]interface{}))[checkField].(string)
			if inName == expectedValue {
				if oResp["metadata"] != nil {
					return (oResp["metadata"].(map[string]interface{}))["guid"].(string), nil
				}
			} else {
				return "", fmt.Errorf("Field %s does not match requested value %s", oResp[checkField], expectedValue)
			}
		} else {
			return "", fmt.Errorf("Warning unknown answer received")
		}
	}

	return "", nil
}

func restoreSharedDomain(sharedDomain SharedDomain) (string, error) {
	showInfo(fmt.Sprintf("Restoring shared domain: %s", sharedDomain.Name))
	oJson, err := json.Marshal(sharedDomain)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/shared_domains", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create shared domain %s, exception message: %s",
			sharedDomain.Name, err.Error()))
	}
	result, err := getResult(resp, "name", sharedDomain.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error restoring shared domain %s: %s", sharedDomain.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully restored shared domain %s", sharedDomain.Name))
	}
	return result, nil
}

func restoreFromJSON(includeSecurityGroups bool) {

	//map["old_guid"] = "new_guid"
	spaceGuids := make(map[string]string)

	var fileContent []byte
	_, err := os.Stat(BackupFile)
	util.FreakOut(err)

	fileContent, err = ioutil.ReadFile(BackupFile)
	util.FreakOut(err)

	backupObject, err := util.ReadBackupJSON(fileContent)
	util.FreakOut(err)

	ccResources := util.CreateSharedDomainsCCResources(nil)
	sharedDomains := ccResources.TransformToResourceModels(backupObject.SharedDomains)

	for _, sd := range *sharedDomains {
		sharedDomain := SharedDomain{Name: sd.Entity["name"].(string)}
		restoreSharedDomain(sharedDomain)
	}

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

			privateDomains := org.Entity["private_domains"].(*[]*models.ResourceModel)
			for _, privateDomain := range *privateDomains {
				pd := PrivateDomain{Name: privateDomain.Entity["name"].(string), OwningOrganizationGuid: org_guid}
				restorePrivateDomain(pd)
			}

			spaces := org.Entity["spaces"].(*[]*models.ResourceModel)
			for _, space := range *spaces {
				s := Space{Name: space.Entity["name"].(string), OrganizationGuid: org_guid}
				space_guid := restoreSpace(s)
				spaceGuids[space.Metadata["guid"].(string)] = space_guid

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

				appsCount := len(*apps)
				appIndex := 1

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

					showInfo(fmt.Sprintf("Restoring App %s for space %s [%d/%d]", a.Name, space.Entity["name"].(string), appIndex, appsCount))

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

						r := Route{
							SpaceGuid: space_guid,
							Port:      route.Entity["port"],
							Path:      route.Entity["path"],
							Host:      route.Entity["host"],
						}

						domainName := domain.Entity["name"].(string)

						if domain.Entity["owning_organization_guid"] == nil {
							domainGuid := getSharedDomainGuid(domainName)
							if domainGuid == "" {
								showWarning(fmt.Sprintf("Could not find shared domain %s", domainName))
								continue
							}
							r.DomainGuid = domainGuid
						} else {
							domainGuid := getPrivateDomainGuid(domainName)
							if domainGuid == "" {
								showWarning(fmt.Sprintf("Could not find private domain %s", domainName))
								continue
							}
							r.DomainGuid = domainGuid
						}
						routeGuid := createRoute(r)
						showInfo(fmt.Sprintf("Binding route %s.%s to app %s", r.Host, domainName, a.Name))
						err = bindRoute(appGuid, routeGuid)
						if err != nil {
							showWarning(fmt.Sprintf("Error binding route %s.%s to app %s: %s", r.Host, domainName, a.Name, err.Error()))
						} else {
							showInfo(fmt.Sprintf("Successfully bound route %s.%s to app %s", r.Host, domainName, a.Name))
						}
					}
					appIndex++
				}
			}
		}
	}

	if includeSecurityGroups {
		ccResources := util.CreateSecurityGroupsCCResources(nil)
		securityGroups := ccResources.TransformToResourceModels(backupObject.SecurityGroups)
		for _, sg := range *securityGroups {
			spaces := *sg.Entity["spaces"].(*[]*models.ResourceModel)
			newSpaces := make([]string, len(spaces))
			for i, s := range spaces {
				newSpaces[i] = spaceGuids[(s.Metadata["guid"]).(string)]
			}

			g := SecurityGroup{
				Name:       sg.Entity["name"].(string),
				Rules:      sg.Entity["rules"],
				SpaceGuids: newSpaces,
			}

			_, err = restoreSecurityGroup(g)
			if err != nil {
				showWarning(fmt.Sprintf("Error restoring security group %s: %s", g.Name, err.Error()))
			}
		}
	}
}

func restoreSecurityGroup(securityGroup SecurityGroup) (string, error) {
	showInfo(fmt.Sprintf("Restoring security group %s", securityGroup.Name))
	resources := util.GetResources(CliConnection, "/v2/security_groups?q=name:"+securityGroup.Name, 1)
	for _, u := range *resources {
		if u.Entity["name"].(string) == securityGroup.Name {
			showInfo(fmt.Sprintf("Deleting old security group %s", securityGroup.Name))
			err := deleteSecurityGroup(u.Metadata["guid"].(string))
			if err != nil {
				return "", err
			} else {
				break
			}
		}
	}
	return createSecurityGroup(securityGroup)
}

func deleteSecurityGroup(guid string) error {
	_, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/security_groups/"+guid, "-H", "Content-Type: application/x-www-form-urlencoded",
		"-X", "DELETE")
	if err != nil {
		return err
	}

	return nil
}

func createSecurityGroup(securityGroup SecurityGroup) (string, error) {
	showInfo(fmt.Sprintf("Creating security group: %s", securityGroup.Name))
	oJson, err := json.Marshal(securityGroup)
	util.FreakOut(err)

	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/security_groups", "-H", "Content-Type: application/json",
		"-d", string(oJson), "-X", "POST")
	if err != nil {
		showWarning(fmt.Sprintf("Could not create security group %s, exception message: %s",
			securityGroup.Name, err.Error()))
	}
	result, err := getResult(resp, "name", securityGroup.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error restoring security group %s: %s", securityGroup.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully restored security group %s", securityGroup.Name))
	}
	return result, nil
}

func bindRoute(appGuid, routeGuid string) error {
	resp, err := CliConnection.CliCommandWithoutTerminalOutput("curl",
		"/v2/apps/"+appGuid+"/routes/"+routeGuid, "-H", "Content-Type: application/x-www-form-urlencoded",
		"-X", "PUT")
	if err != nil {
		return err
	}
	_, err = getResult(resp, "", "")
	if err != nil {
		return err
	}
	return nil
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
	result, err := getResult(resp, "host", route.Host.(string))
	if err != nil {
		showWarning(fmt.Sprintf("Error creating route %s: %s", route.Host, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully created route %s", route.Host))
	}
	return result
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

func getPrivateDomainGuid(domainName string) string {
	resources := util.GetResources(CliConnection, "/v2/private_domains?q=name:"+domainName, 1)
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
	_, err = getResult(resp, "name", app.Name)
	if err != nil {
		showWarning(fmt.Sprintf("Error updating application %s: %s", app.Name, err.Error()))
	} else {
		showInfo(fmt.Sprintf("Succesfully updated application %s", app.Name))
	}
}

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore the CloudFoundry state from a backup",
	Long: `Restore the CloudFoundry state from a backup created with the snapshot command
`,
	Run: func(cmd *cobra.Command, args []string) {
		includeSecurityGroups, _ := cmd.Flags().GetBool("include-security-groups")
		restoreFromJSON(includeSecurityGroups)
	},
}

func init() {
	restoreCmd.Flags().Bool("include-security-groups", false, "Restore security groups")
	RootCmd.AddCommand(restoreCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// restoreCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// restoreCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
