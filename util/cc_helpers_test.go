package util_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hpcloud/cf-plugin-backup/models"
	"github.com/hpcloud/cf-plugin-backup/util"
)

type CCApiMock struct {
	Responses map[string]string
}

func (ccApi *CCApiMock) InvokeGet(url string) (string, error) {
	response, found := ccApi.Responses[url]
	if !found {
		return "", fmt.Errorf("CC URL not found: %v", url)
	}

	return response, nil
}

var fakeRecursiveResponses map[string]string = map[string]string{
	"/v2/organizations": `
{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "metadata": {
            "guid": "91656f3b-0e8d-4cea-9555-4460d309937a",
            "url": "/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a",
            "created_at": "2016-06-17T09:09:44Z",
            "updated_at": null
         },
         "entity": {
            "name": "o1",
            "billing_enabled": false,
            "quota_definition_guid": "8d331df5-4bea-4116-b64b-f9c90c3c14bb",
            "status": "active",
            "spaces_url": "/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a/spaces"
         }
      }
   ]
}
`,

	"/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a/spaces": `
{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "metadata": {
            "guid": "fac8c0f5-0e48-4a1c-a8ef-13aae586a650",
            "url": "/v2/spaces/fac8c0f5-0e48-4a1c-a8ef-13aae586a650",
            "created_at": "2016-06-17T09:09:52Z",
            "updated_at": null
         },
         "entity": {
            "name": "s1",
            "organization_guid": "91656f3b-0e8d-4cea-9555-4460d309937a",
            "space_quota_definition_guid": null,
            "allow_ssh": true,
            "organization_url": "/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a"
         }
      }
   ]
}
`,

	"/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a": `
{
   "metadata": {
      "guid": "91656f3b-0e8d-4cea-9555-4460d309937a",
      "url": "/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a",
      "created_at": "2016-06-17T09:09:44Z",
      "updated_at": null
   },
   "entity": {
      "name": "o1",
      "billing_enabled": false,
      "quota_definition_guid": "8d331df5-4bea-4116-b64b-f9c90c3c14bb",
      "status": "active",
      "spaces_url": "/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a/spaces"
   }
}
`,
}

func TestGetResources_EmptyOrgs(t *testing.T) {
	fakeResponses := map[string]string{
		"/v2/organizations": `
		{
		   "total_results": 1,
		   "total_pages": 1,
		   "prev_url": null,
		   "next_url": null,
		   "resources": []
		}
		`,
	}
	ccApi := CCApiMock{Responses: fakeResponses}

	result, err := util.GetOrgsResourcesRecurively(&ccApi)
	if err != nil {
		t.Fatal(err)
	}

	if len(*result) != 0 {
		t.Fatal("result is not of length 0")
	}
}

func TestGetResources_OneOrg(t *testing.T) {
	fakeResponses := map[string]string{
		"/v2/organizations": `
{
   "total_results": 1,
   "total_pages": 1,
   "prev_url": null,
   "next_url": null,
   "resources": [
      {
         "metadata": {
            "guid": "91656f3b-0e8d-4cea-9555-4460d309937a",
            "url": "/v2/organizations/91656f3b-0e8d-4cea-9555-4460d309937a",
            "created_at": "2016-06-17T09:09:44Z",
            "updated_at": null
         },
         "entity": {
            "name": "o1",
            "billing_enabled": false,
            "quota_definition_guid": "8d331df5-4bea-4116-b64b-f9c90c3c14bb",
            "status": "active"
         }
      }
   ]
}
		`,
	}
	ccApi := CCApiMock{Responses: fakeResponses}

	result, err := util.GetOrgsResourcesRecurively(&ccApi)
	if err != nil {
		t.Fatal(err)
	}

	if len(*result) != 1 {
		t.Fatal("result is not of length 1")
	}

	if (*(*result)[0]).Entity["name"] != "o1" {
		t.Fatal("oranization name not equal to o1")
	}
}

func TestGetResources_RecurseWithLoops(t *testing.T) {
	ccApi := CCApiMock{Responses: fakeRecursiveResponses}

	ccResources := util.CreateOrgCCResources(&ccApi)
	result := ccResources.GetResources(util.OrgsURL, 10)

	if len(*result) != 1 {
		t.Fatal("result is not of length 1")
	}

	o1org := *(*result)[0]

	if o1org.Entity["name"] != "o1" {
		t.Fatal("oranization name not equal to o1")
	}

	spaces := *(o1org.Entity["spaces"]).(*[]*models.ResourceModel)

	if len(spaces) != 1 {
		t.Fatal("spaces are missing")
	}

	if spaces[0].Entity["name"] != "s1" {
		t.Fatal("invalid space name")
	}

	orgRefFromSpace := spaces[0].Entity["organization"].(*models.ResourceModel)

	if orgRefFromSpace.Entity["name"] != "o1" {
		t.Fatal("invalid org name refrenced from space")
	}
}

func TestGetResources_StacksArePulled(t *testing.T) {
	fakeResponses := map[string]string{}
	err := json.Unmarshal([]byte(ccRecording1), &fakeResponses)

	if err != nil {
		t.Fatal("json.Unmarshal failed", err)
	}
	ccApi := CCApiMock{Responses: fakeResponses}

	result, err := util.GetOrgsResourcesRecurively(&ccApi)
	if err != nil {
		t.Fatal("GetOrgsResourcesRecurively failed", err)
	}

	if len(*result) != 2 {
		t.Fatal("result is not of length 1")
	}

	o1org := *(*result)[0]

	if o1org.Entity["name"] != "o" {
		t.Fatal("oranization name not equal to o")
	}

	spaces := *(o1org.Entity["spaces"]).(*[]*models.ResourceModel)

	if len(spaces) != 1 {
		t.Fatal("spaces are missing")
	}

	if spaces[0].Entity["name"] != "s" {
		t.Fatal("invalid space name")
	}

	sapps := *(spaces[0].Entity["apps"].(*[]*models.ResourceModel))
	if len(sapps) != 4 {
		t.Fatal("apps are missing, expected 4 found ", len(sapps))
	}

	for _, iapp := range sapps {
		if iapp.Entity["stack"] == nil && iapp.Entity["stack_url"] != nil {
			t.Fatal("stack is missing for ", iapp.Entity["name"].(string), iapp)
		}
	}
}
