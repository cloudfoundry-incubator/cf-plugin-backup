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

type FollowDecision func(childKey string) bool

type CCApi interface {
	InvokeGet(path string) (string, error)
}

type CliConnectionCCApi struct {
	CliConnection plugin.CliConnection
}

func (ccApi *CliConnectionCCApi) InvokeGet(path string) (string, error) {
	output, err := ccApi.CliConnection.CliCommandWithoutTerminalOutput("curl", path, "-X", "GET")
	return output[0], err
}

type CCResources struct {
	ccApi CCApi

	retriveCache   map[string]interface{}
	transformCache map[string]interface{}

	follow FollowDecision
}

func NewCCResources(ccApi CCApi, follow FollowDecision) *CCResources {
	res := CCResources{}
	res.ccApi = ccApi
	res.follow = follow

	res.retriveCache = make(map[string]interface{})
	res.transformCache = make(map[string]interface{})

	return &res
}

func (ccResources *CCResources) FollowRelation(resource *models.ResourceModel, relationsDepth int) {
	if relationsDepth > 0 {
		for k, childURL := range resource.Entity {
			if strings.HasSuffix(k, UrlSuffix) {
				childKey := strings.TrimSuffix(k, UrlSuffix)
				if _, ok := resource.Entity[childKey]; !ok {
					if ccResources.follow == nil || ccResources.follow(childKey) {
						childResource := ccResources.GetGenericResource(childURL.(string), relationsDepth-1)
						resource.Entity[childKey] = childResource
					}
				}
			}
		}
	}
}

func (ccResources *CCResources) GetGenericResource(url string, relationsDepth int) interface{} {
	if cacheEntry, hit := ccResources.retriveCache[url]; hit {
		return cacheEntry
	}

	output, err := ccResources.ccApi.InvokeGet(url)
	FreakOut(err)

	var result map[string]interface{}

	err = json.Unmarshal([]byte(output), &result)
	FreakOut(err)

	if _, ok := result["total_results"]; ok {
		return ccResources.GetResources(url, relationsDepth)
	}

	return ccResources.GetResource(url, relationsDepth)
}

func (ccResources *CCResources) GetResource(url string, relationsDepth int) *models.ResourceModel {
	if cacheEntry, hit := ccResources.retriveCache[url]; hit {
		return cacheEntry.(*models.ResourceModel)
	}

	log.Println("Retrinving resource", url)

	output, err := ccResources.ccApi.InvokeGet(url)
	FreakOut(err)

	resource := &models.ResourceModel{}

	err = json.Unmarshal([]byte(output), &resource)
	FreakOut(err)

	ccResources.retriveCache[resource.Metadata["url"].(string)] = resource

	ccResources.FollowRelation(resource, relationsDepth)

	return resource
}

func (ccResources *CCResources) GetResources(url string, relationsDepth int) *[]*models.ResourceModel {
	if cacheEntry, hit := ccResources.retriveCache[url]; hit {
		return cacheEntry.(*[]*models.ResourceModel)
	}

	log.Println("Retrinving resources from", url)

	nextURL := url
	allResources := make([]*models.ResourceModel, 0)

	for nextURL != "" {
		collectionModel := models.ResourceCollectionModel{}

		output, err := ccResources.ccApi.InvokeGet(nextURL)
		FreakOut(err)

		err = json.Unmarshal([]byte(output), &collectionModel)
		FreakOut(err)

		log.Printf("Retrived %v/%v from %v", len(allResources)+len(*collectionModel.Resources), collectionModel.TotalResults, url)

		for _, resource := range *collectionModel.Resources {
			if cacheEntry, hit := ccResources.retriveCache[resource.Metadata["url"].(string)]; hit {
				resource = cacheEntry.(*models.ResourceModel)
			} else {
				ccResources.retriveCache[resource.Metadata["url"].(string)] = resource
			}

			allResources = append(allResources, resource)
			ccResources.FollowRelation(resource, relationsDepth)
		}

		nextURL = collectionModel.NextURL
	}

	ccResources.retriveCache[url] = &allResources

	return &allResources
}

func BreakResourceLoops(resources *[]*models.ResourceModel) *[]*models.ResourceModel {
	var result []*models.ResourceModel

	visitedResourceMap := make(map[interface{}]bool)
	queue := list.New()

	for _, resource := range *resources {
		visitedResourceMap[resource] = true

		resourceCopy := *resource
		result = append(result, &resourceCopy)

		queue.PushBack(&resourceCopy)
	}

	for {
		e := queue.Front()
		if e == nil {
			break
		}

		resource := e.Value.(*models.ResourceModel)

		for childKey, childValue := range resource.Entity {
			// Single value
			singleChildResource, isResourceModel := childValue.(*models.ResourceModel)
			if isResourceModel {
				if _, visited := visitedResourceMap[singleChildResource]; !visited {
					visitedResourceMap[singleChildResource] = true

					copyValue := *singleChildResource
					resource.Entity[childKey] = &copyValue

					queue.PushBack(&copyValue)
				} else {
					delete(resource.Entity, childKey)
				}

			}

			// Array
			multipleChildResources, isResourceModel := childValue.(*[]*models.ResourceModel)
			if isResourceModel {
				if _, visited := visitedResourceMap[multipleChildResources]; !visited {
					visitedResourceMap[multipleChildResources] = true

					var multipleChildResourcesCopy []*models.ResourceModel

					for _, childResource := range *multipleChildResources {
						childResourceCopy := *childResource
						multipleChildResourcesCopy = append(multipleChildResourcesCopy, &childResourceCopy)

						if _, visited := visitedResourceMap[childResource]; !visited {
							visitedResourceMap[childResource] = true
							queue.PushBack(&childResourceCopy)
						} else {
							childResourceCopy.Entity = nil
						}

						resource.Entity[childKey] = &multipleChildResourcesCopy
					}
				} else {
					delete(resource.Entity, childKey)
				}
			}
		}

		queue.Remove(e)
	}

	return &result
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
				childResource := ccResources.TransformToResourceGeneric(childEntity)

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

func (ccResources *CCResources) TransformToResourceGeneric(r interface{}) interface{} {
	switch r.(type) {
	case map[string]interface{}:
		return ccResources.TransformToResource(r)
	case []interface{}:
		return ccResources.TransformToResources(r)
	}

	panic("unknown resource type")
}

func (ccResources *CCResources) TransformToResource(resource interface{}) *models.ResourceModel {
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

func (ccResources *CCResources) TransformToResources(resources interface{}) *[]*models.ResourceModel {
	var result []*models.ResourceModel

	resourceArray := resources.([]interface{})

	for _, r := range resourceArray {
		resourceModel := ccResources.TransformToResource(r)
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

		"service_instances",
		"service_instance",

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
	resources := ccResources.GetResources(OrgsUrl, 10)

	resources = BreakResourceLoops(resources)

	return resources, nil
}

func CreateSharedDomainsCCResources(ccApi CCApi) *CCResources {
	follow := func(childKey string) bool {
		return false
	}

	ccResources := NewCCResources(ccApi, follow)

	return ccResources
}

func RestoreOrgResourceModels(orgResources interface{}) *[]*models.ResourceModel {
	ccResources := CreateOrgCCResources(nil)
	transformedRes := ccResources.TransformToResources(orgResources)

	return transformedRes
}

func GetSharedDomains(ccApi CCApi) (interface{}, error) {
	ccResources := CreateSharedDomainsCCResources(ccApi)

	resources := ccResources.GetResources(ShardDomainsUrl, 1)
	resources = BreakResourceLoops(resources)

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
