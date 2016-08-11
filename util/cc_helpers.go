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

const urlSuffix string = "_url"

// OrgsURL represents the organizations url path
const OrgsURL = "/v2/organizations"
const shardDomainsURL = "/v2/shared_domains"
const securityGroupsURL = "/v2/security_groups"
const featureFlagsURL = "/v2/config/feature_flags"

type followDecision func(childKey string) bool

type cCApi interface {
	InvokeGet(path string) (string, error)
}

//CliConnectionCCApi represents the cf cli connection
type CliConnectionCCApi struct {
	CliConnection plugin.CliConnection
}

// InvokeGet invokes GET on a given path
func (ccAPI *CliConnectionCCApi) InvokeGet(path string) (string, error) {
	output, err := ccAPI.CliConnection.CliCommandWithoutTerminalOutput("curl", path, "-X", "GET")

	return ConcatStringArray(output), err
}

// CCResources represents cc resources
type CCResources struct {
	ccAPI cCApi

	jsonRetriveCache map[string][]byte
	transformCache   map[string]interface{}

	follow followDecision
}

func newCCResources(ccAPI cCApi, follow followDecision) *CCResources {
	res := CCResources{}
	res.ccAPI = ccAPI
	res.follow = follow

	res.jsonRetriveCache = make(map[string][]byte)
	res.transformCache = make(map[string]interface{})

	return &res
}

func (ccResources *CCResources) retriveJSONGenericResource(url string) ([]byte, error) {
	var result []byte

	if cacheResult, cacheHit := ccResources.jsonRetriveCache[url]; cacheHit {
		result = cacheResult
	} else {
		log.Println("Retrieving resource", url)

		output, err := ccResources.ccAPI.InvokeGet(url)
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
			var allResources []interface{}

			for nextURL != "" {
				output, err := ccResources.ccAPI.InvokeGet(nextURL)
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
				elementURL := metadata.(map[string]interface{})["url"].(string)
				jsonElement, err := json.Marshal(element)
				if err != nil {
					return nil, err
				}

				ccResources.jsonRetriveCache[elementURL] = jsonElement
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
	jsonOutput, err := ccResources.retriveJSONGenericResource(url)
	if err != nil {
		return nil, err
	}

	var collection *models.ResourceCollectionModel

	err = json.Unmarshal([]byte(jsonOutput), &collection)
	if err == nil && collection.TotalPages > 0 {
		if collection.Resources != nil {
			return collection.Resources, nil
		}
		var emptyResult []*models.ResourceModel
		return &emptyResult, nil
	}

	var singleValue *models.ResourceModel
	err = json.Unmarshal([]byte(jsonOutput), &singleValue)
	if err == nil {
		return singleValue, nil
	}

	return nil, err
}

func (ccResources *CCResources) retriveFeatureFlagsResource(url string) ([]*models.FeatureFlagModel, error) {

	log.Println("Retrieving resource", url)

	output, err := ccResources.ccAPI.InvokeGet(url)
	if err != nil {
		return nil, err
	}

	var collection []*models.FeatureFlagModel

	err = json.Unmarshal([]byte(output), &collection)

	return collection, err
}

type retriveQueueItem struct {
	resourceTarget interface{}
	depth          int
}

func (ccResources *CCResources) getGenericResource(startURL string, relationsDepth int) interface{} {
	startResource, err := ccResources.retriveParsedGenericResource(startURL)
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
					if strings.HasSuffix(entityKey, urlSuffix) {
						childEntity := strings.TrimSuffix(entityKey, urlSuffix)

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

// GetResource gets a resource
func (ccResources *CCResources) GetResource(url string, relationsDepth int) *models.ResourceModel {
	res := ccResources.getGenericResource(url, relationsDepth)
	return res.(*models.ResourceModel)
}

// GetResources gets resources
func (ccResources *CCResources) GetResources(url string, relationsDepth int) *[]*models.ResourceModel {
	res := ccResources.getGenericResource(url, relationsDepth)
	return res.(*[]*models.ResourceModel)
}

func (ccResources *CCResources) recreateLinkForEntity(resource *models.ResourceModel) {
	for k, v := range resource.Entity {
		if strings.HasSuffix(k, urlSuffix) {
			childURL, isValidURLEntry := v.(string)
			if !isValidURLEntry && childURL != "" {
				continue
			}

			childKey := strings.TrimSuffix(k, urlSuffix)

			if !(ccResources.follow == nil || ccResources.follow(childKey)) {
				continue
			}

			if childEntity, hasEntity := resource.Entity[childKey]; hasEntity {
				if childEntity == nil {
					continue
				}

				childResource := ccResources.transformToResourceModelGeneric(childEntity)

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

func (ccResources *CCResources) transformToResourceModelGeneric(r interface{}) interface{} {
	switch r.(type) {
	case map[string]interface{}:
		return ccResources.transformToResourceModel(r)
	case []interface{}:
		return ccResources.TransformToResourceModels(r)
	}
	log.Fatalf("unknown resource type %T", r)
	return nil
}

func (ccResources *CCResources) transformToResourceModel(resource interface{}) *models.ResourceModel {
	resourceModel := models.ResourceModel{}

	resourceValue := resource.(map[string]interface{})

	resourceModel.Metadata = resourceValue["metadata"].(map[string]interface{})
	entity, hasEntity := resourceValue["entity"].(map[string]interface{})

	// Cache handling

	resourceURL := resourceModel.Metadata["url"].(string)
	if cacheEntry, hit := ccResources.transformCache[resourceURL]; hit {
		cacheEntryValue := cacheEntry.(*models.ResourceModel)

		cacheEntryHasEntity := cacheEntryValue.Entity != nil
		if hasEntity && !cacheEntryHasEntity {
			cacheEntryValue.Entity = resourceModel.Entity
		}
		if !hasEntity && cacheEntryHasEntity {
			resourceModel.Entity = cacheEntryValue.Entity
		}
	} else {
		ccResources.transformCache[resourceURL] = &resourceModel
	}

	if hasEntity {
		resourceModel.Entity = entity
		ccResources.recreateLinkForEntity(&resourceModel)
	}

	return &resourceModel
}

// TransformToResourceModels transforms interface to resource models
func (ccResources *CCResources) TransformToResourceModels(resources interface{}) *[]*models.ResourceModel {
	var result []*models.ResourceModel

	resourceArray := resources.([]interface{})

	for _, r := range resourceArray {
		resourceModel := ccResources.transformToResourceModel(r)
		result = append(result, resourceModel)
	}

	return &result
}

func (ccResources *CCResources) transformToFlagModels(resources interface{}) *[]*models.FeatureFlagModel {
	var result []*models.FeatureFlagModel

	resourceArray := resources.([]interface{})

	for _, r := range resourceArray {
		resourceModel := r.(map[string]interface{})
		var flag models.FeatureFlagModel

		flag.Name = resourceModel["name"].(string)
		flag.Enabled = resourceModel["enabled"].(bool)
		flag.URL = resourceModel["Url"].(string)

		if val, ok := resourceModel["error_message"]; ok {
			flag.ErrorMessage = val.(string)
		}

		result = append(result, &flag)

	}

	return &result
}

// CreateOrgCCResources creates org resource
func CreateOrgCCResources(ccAPI cCApi) *CCResources {
	resourceURLsWhitelistSlice := []interface{}{
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
	resourceURLsWhitelist := mapset.NewSetFromSlice(resourceURLsWhitelistSlice)

	follow := func(childKey string) bool {
		return resourceURLsWhitelist.Contains(childKey)
	}

	ccResources := newCCResources(ccAPI, follow)

	return ccResources
}

// GetResources retrieves resources for a given url
func GetResources(cliConnection plugin.CliConnection, url string, relationsDepth int) *[]*models.ResourceModel {
	follow := func(childKey string) bool {
		return false
	}

	ccResources := newCCResources(&CliConnectionCCApi{CliConnection: cliConnection}, follow)

	resources := ccResources.GetResources(url, relationsDepth)

	return resources
}

// CreateFeatureFlagsCCResources creates feature flags resources
func CreateFeatureFlagsCCResources(ccAPI cCApi) *CCResources {
	follow := func(childKey string) bool {
		return false
	}

	ccResources := newCCResources(ccAPI, follow)

	return ccResources
}

// GetOrgsResourcesRecurively returns all orgs
func GetOrgsResourcesRecurively(ccAPI cCApi) (*[]*models.ResourceModel, error) {
	ccResources := CreateOrgCCResources(ccAPI)
	resources := ccResources.GetResources(OrgsURL, 5)

	return resources, nil
}

// CreateSharedDomainsCCResources creates shared domains resources
func CreateSharedDomainsCCResources(ccAPI cCApi) *CCResources {
	follow := func(childKey string) bool {
		return false
	}

	ccResources := newCCResources(ccAPI, follow)

	return ccResources
}

// CreateSecurityGroupsCCResources creates security groups resources
func CreateSecurityGroupsCCResources(ccAPI cCApi) *CCResources {
	resourceURLsWhitelistSlice := []interface{}{
		"spaces",
		"organization",
	}
	resourceURLsWhitelist := mapset.NewSetFromSlice(resourceURLsWhitelistSlice)

	follow := func(childKey string) bool {
		return resourceURLsWhitelist.Contains(childKey)
	}

	ccResources := newCCResources(ccAPI, follow)

	return ccResources
}

// RestoreOrgResourceModels gets orgs as resource models
func RestoreOrgResourceModels(orgResources interface{}) *[]*models.ResourceModel {
	ccResources := CreateOrgCCResources(nil)
	transformedRes := ccResources.TransformToResourceModels(orgResources)

	return transformedRes
}

// RestoreFlagsResourceModels gets flags as resource models
func RestoreFlagsResourceModels(flagResources interface{}) *[]*models.FeatureFlagModel {
	ccResources := CreateFeatureFlagsCCResources(nil)
	result := ccResources.transformToFlagModels(flagResources)

	return result
}

// GetSharedDomains returns shared domains
func GetSharedDomains(ccAPI cCApi) (interface{}, error) {
	ccResources := CreateSharedDomainsCCResources(ccAPI)

	resources := ccResources.GetResources(shardDomainsURL, 1)

	return resources, nil
}

// GetFeatureFlags returns feature flags
func GetFeatureFlags(ccAPI cCApi) (*[]*models.FeatureFlagModel, error) {

	ccResources := CreateFeatureFlagsCCResources(ccAPI)

	ccFlagsResources, err := ccResources.retriveFeatureFlagsResource(featureFlagsURL)

	if err != nil {
		return nil, err
	}

	return &ccFlagsResources, nil
}

// GetSecurityGroups return security groups
func GetSecurityGroups(ccAPI cCApi) (interface{}, error) {
	ccResources := CreateSecurityGroupsCCResources(ccAPI)

	resources := ccResources.GetResources(securityGroupsURL, 2)

	return resources, nil
}

// CreateBackupJSON creates backup json
func CreateBackupJSON(backupModel models.BackupModel) (string, error) {
	jsonResources, err := json.MarshalIndent(backupModel, "", " ")
	if err != nil {
		return "", err
	}

	return string(jsonResources), nil
}

// ReadBackupJSON reads backup json
func ReadBackupJSON(jsonBytes []byte) (*models.BackupModel, error) {
	backupModel := models.BackupModel{}
	err := json.Unmarshal([]byte(jsonBytes), &backupModel)
	if err != nil {
		return nil, err
	}

	return &backupModel, nil
}
