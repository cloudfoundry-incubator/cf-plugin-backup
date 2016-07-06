package util

import (
	"container/list"
	"encoding/json"
	"log"
	"strings"

	"github.com/deckarep/golang-set" // MIT License

	"github.com/cloudfoundry/cli/plugin"
	"github.com/hpcloud/cf-plugin-backup/models"
)

const UrlSuffix string = "_url"
const OrgsUrl = "/v2/organizations"
const ShardDomainsUrl = "/v2/shared_domains"
const SecurityGroupsUrl = "/v2/security_groups"

type FollowDecision func(childKey string) bool

type CCApi interface {
	InvokeGet(path string) (string, error)
}

type CliConnectionCCApi struct {
	CliConnection plugin.CliConnection
}

func (ccApi *CliConnectionCCApi) InvokeGet(path string) (string, error) {
	output, err := ccApi.CliConnection.CliCommandWithoutTerminalOutput("curl", path, "-X", "GET")

	return ConcatStringArray(output), err
}

type CCResources struct {
	ccApi CCApi

	jsonRetriveCache map[string][]byte
	transformCache   map[string]interface{}

	follow FollowDecision
}

func NewCCResources(ccApi CCApi, follow FollowDecision) *CCResources {
	res := CCResources{}
	res.ccApi = ccApi
	res.follow = follow

	res.jsonRetriveCache = make(map[string][]byte)
	res.transformCache = make(map[string]interface{})

	return &res
}

func (ccResources *CCResources) retriveJsonGenericResource(url string) ([]byte, error) {
	var result []byte

	if cacheResult, cacheHit := ccResources.jsonRetriveCache[url]; cacheHit {
		result = cacheResult
	} else {
		log.Println("Retrieving resource", url)

		output, err := ccResources.ccApi.InvokeGet(url)
		if err != nil {
			return nil, err
		}

		var parsedOutput map[string]interface{}

		err = json.Unmarshal([]byte(output), &parsedOutput)
		if err != nil {
			return nil, err
		}

		if _, isArray := parsedOutput["total_results"]; isArray {
			nextURL := url
			allResources := make([]interface{}, 0)

			for nextURL != "" {
				output, err := ccResources.ccApi.InvokeGet(nextURL)
				if err != nil {
					return nil, err
				}

				var collection map[string]interface{}
				err = json.Unmarshal([]byte(output), &collection)
				if err != nil {
					return nil, err
				}

				allResources = append(allResources, collection["resources"].([]interface{})...)

				if collection["next_url"] != nil {
					nextURL = collection["next_url"].(string)
				} else {
					nextURL = ""
				}
			}

			parsedOutput["next_url"] = nil
			parsedOutput["prev_url"] = nil
			parsedOutput["total_pages"] = 1
			parsedOutput["total_results"] = len(allResources)
			parsedOutput["resources"] = allResources

			for _, element := range allResources {
				metadata := element.(map[string]interface{})["metadata"]
				elementUrl := metadata.(map[string]interface{})["url"].(string)
				jsonElement, err := json.Marshal(element)
				if err != nil {
					return nil, err
				}

				ccResources.jsonRetriveCache[elementUrl] = jsonElement
			}
		}

		jsonOutput, err := json.Marshal(parsedOutput)
		if err != nil {
			return nil, err
		}

		ccResources.jsonRetriveCache[url] = jsonOutput
		result = jsonOutput
	}

	return result, nil
}

func (ccResources *CCResources) retriveParsedGenericResource(url string) (interface{}, error) {
	jsonOutput, err := ccResources.retriveJsonGenericResource(url)
	if err != nil {
		return nil, err
	}

	var collection *models.ResourceCollectionModel

	err = json.Unmarshal([]byte(jsonOutput), &collection)
	if err == nil && collection.TotalPages > 0 {
		if collection.Resources != nil {
			return collection.Resources, nil
		} else {
			emptyResult := make([]*models.ResourceModel, 0)
			return &emptyResult, nil
		}
	}

	var singleValue *models.ResourceModel
	err = json.Unmarshal([]byte(jsonOutput), &singleValue)
	if err == nil {
		return singleValue, nil
	}

	return nil, err
}

type retriveQueueItem struct {
	resourceTarget interface{}
	depth          int
}

func (ccResources *CCResources) GetGenericResource(startUrl string, relationsDepth int) interface{} {
	startResource, err := ccResources.retriveParsedGenericResource(startUrl)
	FreakOut(err)

	queue := list.New()
	queue.PushBack(retriveQueueItem{resourceTarget: startResource, depth: 0})

	for {
		e := queue.Front()
		if e == nil {
			break
		}

		queueElement := e.Value.(retriveQueueItem)
		currentDepth := queueElement.depth

		if currentDepth < relationsDepth {

			if listOfResources, isArray := queueElement.resourceTarget.(*[]*models.ResourceModel); isArray {
				for _, resource := range *listOfResources {
					queue.PushBack(retriveQueueItem{resourceTarget: resource, depth: currentDepth})
				}
			} else {
				resource := queueElement.resourceTarget.(*models.ResourceModel)

				for entityKey, entityValue := range resource.Entity {
					if strings.HasSuffix(entityKey, UrlSuffix) {
						childEntity := strings.TrimSuffix(entityKey, UrlSuffix)

						if ccResources.follow == nil || ccResources.follow(childEntity) {
							childURL := entityValue.(string)
							childResource, err := ccResources.retriveParsedGenericResource(childURL)
							FreakOut(err)

							resource.Entity[childEntity] = childResource
							queue.PushBack(retriveQueueItem{resourceTarget: childResource, depth: currentDepth + 1})
						}
					}
				}

			}
		}

		queue.Remove(e)
	}

	return startResource
}

func (ccResources *CCResources) GetResource(url string, relationsDepth int) *models.ResourceModel {
	res := ccResources.GetGenericResource(url, relationsDepth)
	return res.(*models.ResourceModel)
}

func (ccResources *CCResources) GetResources(url string, relationsDepth int) *[]*models.ResourceModel {
	res := ccResources.GetGenericResource(url, relationsDepth)
	return res.(*[]*models.ResourceModel)
}

func (ccResources *CCResources) RecreateLinkForEntity(resource *models.ResourceModel) {
	for k, v := range resource.Entity {
		if strings.HasSuffix(k, UrlSuffix) {
			childURL, isValidUrlEntry := v.(string)
			if !isValidUrlEntry && childURL != "" {
				continue
			}

			childKey := strings.TrimSuffix(k, UrlSuffix)

			if !(ccResources.follow == nil || ccResources.follow(childKey)) {
				continue
			}

			if childEntity, hasEntity := resource.Entity[childKey]; hasEntity {
				childResource := ccResources.TransformToResourceModelGeneric(childEntity)

				resource.Entity[childKey] = childResource

				if cacheEntry, hit := ccResources.transformCache[childURL]; hit {
					if resourceCacheEntry, isSingleResource := cacheEntry.(*models.ResourceModel); isSingleResource {
						if resourceCacheEntry.Entity == nil {
							child := childResource.(*models.ResourceModel)
							resourceCacheEntry.Entity = child.Entity
						}
					}
				} else {
					ccResources.transformCache[childURL] = childResource
				}
			} else {
				if cacheEntry, hit := ccResources.transformCache[childURL]; hit {
					resource.Entity[childKey] = cacheEntry
				}
			}
		}
	}
}

func (ccResources *CCResources) TransformToResourceModelGeneric(r interface{}) interface{} {
	switch r.(type) {
	case map[string]interface{}:
		return ccResources.TransformToResourceModel(r)
	case []interface{}:
		return ccResources.TransformToResourceModels(r)
	}

	panic("unknown resource type")
}

func (ccResources *CCResources) TransformToResourceModel(resource interface{}) *models.ResourceModel {
	resourceModel := models.ResourceModel{}

	resourceValue := resource.(map[string]interface{})

	resourceModel.Metadata = resourceValue["metadata"].(map[string]interface{})
	entity, hasEntity := resourceValue["entity"].(map[string]interface{})

	// Cache handling

	resourceUrl := resourceModel.Metadata["url"].(string)
	if cacheEntry, hit := ccResources.transformCache[resourceUrl]; hit {
		cacheEntryValue := cacheEntry.(*models.ResourceModel)

		cacheEntryHasEntity := cacheEntryValue.Entity != nil
		if hasEntity && !cacheEntryHasEntity {
			cacheEntryValue.Entity = resourceModel.Entity
		}
		if !hasEntity && cacheEntryHasEntity {
			resourceModel.Entity = cacheEntryValue.Entity
		}
	} else {
		ccResources.transformCache[resourceUrl] = &resourceModel
	}

	if hasEntity {
		resourceModel.Entity = entity
		ccResources.RecreateLinkForEntity(&resourceModel)
	}

	return &resourceModel
}

func (ccResources *CCResources) TransformToResourceModels(resources interface{}) *[]*models.ResourceModel {
	var result []*models.ResourceModel

	resourceArray := resources.([]interface{})

	for _, r := range resourceArray {
		resourceModel := ccResources.TransformToResourceModel(r)
		result = append(result, resourceModel)
	}

	return &result
}

func CreateOrgCCResources(ccApi CCApi) *CCResources {
	resourceUrlsWhitelistSlice := []interface{}{
		"organizations",
		"organization",
		"auditors", "managers", "billing_managers",
		"quota_definition",

		"spaces",
		"developers", "auditors", "managers",
		"space_quota_definitions",

		"apps",
		"app",

		"routes",
		"route",
		"route_mappings",

		"domains",
		"domain",
		"private_domains",

		"stack",
	}
	resourceUrlsWhitelist := mapset.NewSetFromSlice(resourceUrlsWhitelistSlice)

	follow := func(childKey string) bool {
		return resourceUrlsWhitelist.Contains(childKey)
	}

	ccResources := NewCCResources(ccApi, follow)

	return ccResources
}

func GetResources(cliConnection plugin.CliConnection, url string, relationsDepth int) *[]*models.ResourceModel {
	follow := func(childKey string) bool {
		return false
	}

	ccResources := NewCCResources(&CliConnectionCCApi{CliConnection: cliConnection}, follow)

	resources := ccResources.GetResources(url, relationsDepth)

	return resources
}

func GetOrgsResourcesRecurively(ccApi CCApi) (*[]*models.ResourceModel, error) {
	ccResources := CreateOrgCCResources(ccApi)
	resources := ccResources.GetResources(OrgsUrl, 5)

	return resources, nil
}

func CreateSharedDomainsCCResources(ccApi CCApi) *CCResources {
	follow := func(childKey string) bool {
		return false
	}

	ccResources := NewCCResources(ccApi, follow)

	return ccResources
}

func CreateSecurityGroupsCCResources(ccApi CCApi) *CCResources {
	resourceUrlsWhitelistSlice := []interface{}{
		"spaces",
		"organization",
	}
	resourceUrlsWhitelist := mapset.NewSetFromSlice(resourceUrlsWhitelistSlice)

	follow := func(childKey string) bool {
		return resourceUrlsWhitelist.Contains(childKey)
	}

	ccResources := NewCCResources(ccApi, follow)

	return ccResources
}

func RestoreOrgResourceModels(orgResources interface{}) *[]*models.ResourceModel {
	ccResources := CreateOrgCCResources(nil)
	transformedRes := ccResources.TransformToResourceModels(orgResources)

	return transformedRes
}

func GetSharedDomains(ccApi CCApi) (interface{}, error) {
	ccResources := CreateSharedDomainsCCResources(ccApi)

	resources := ccResources.GetResources(ShardDomainsUrl, 1)

	return resources, nil
}

func GetSecurityGroups(ccApi CCApi) (interface{}, error) {
	ccResources := CreateSecurityGroupsCCResources(ccApi)

	resources := ccResources.GetResources(SecurityGroupsUrl, 2)

	return resources, nil
}

func CreateBackupJSON(backupModel models.BackupModel) (string, error) {
	jsonResources, err := json.MarshalIndent(backupModel, "", " ")
	if err != nil {
		return "", err
	}

	return string(jsonResources), nil
}

func ReadBackupJSON(jsonBytes []byte) (*models.BackupModel, error) {
	backupModel := models.BackupModel{}
	err := json.Unmarshal([]byte(jsonBytes), &backupModel)
	if err != nil {
		return nil, err
	}

	return &backupModel, nil
}
