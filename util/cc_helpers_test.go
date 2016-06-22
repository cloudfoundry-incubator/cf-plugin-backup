package util_test

import (
	"testing"

	"github.com/hpcloud/cf-plugin-backup/util"
)

type CCApiMock struct {
	Responses map[string]string
}

func (ccApi *CCApiMock) InvokeGet(url string) (string, error) {
	return ccApi.Responses[url], nil
}

func TestGetResources_EmptyOrgs(t *testing.T) {
	faceResponses := map[string]string{
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
	ccApi := CCApiMock{Responses: faceResponses}

	result, err := util.GetOrgsResourcesRecurively(&ccApi)
	if err != nil {
		t.Fatal(err)
	}

	if len(*result) != 0 {
		t.Fatal("result is not of length 0")
	}
}

func TestGetResources_OneOrg(t *testing.T) {
	faceResponses := map[string]string{
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
	ccApi := CCApiMock{Responses: faceResponses}

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
