package util

import (
	"container/list"
	"encoding/json"
	"strings"

	"github.com/deckarep/golang-set" // MIT License

	"github.com/cloudfoundry/cli/plugin"
	"github.com/hpcloud/cf-plugin-backup/models"
)

type FollowDecision func(childKey string) bool

func FollowRelation(cliConnection plugin.CliConnection, resource *models.ResourceModel, relationsDepth int, follow FollowDecision, cache map[string]interface{}) {
	if relationsDepth > 0 {
		for k, childURL := range resource.Entity {
			if strings.HasSuffix(k, models.UrlSuffix) {
				childKey := strings.TrimSuffix(k, models.UrlSuffix)
				if _, ok := resource.Entity[childKey]; !ok {
					if follow == nil || follow(childKey) {
						childResource := GetGenericResource(cliConnection, childURL.(string), relationsDepth-1, follow, cache)
						resource.Entity[childKey] = childResource
					}
				}
			}
		}
	}
}

func GetGenericResource(cliConnection plugin.CliConnection, url string, relationsDepth int, follow FollowDecision, cache map[string]interface{}) interface{} {
	output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", url, "-X", "GET")
	FreakOut(err)

	var result map[string]interface{}

	err = json.Unmarshal([]byte(output[0]), &result)
	FreakOut(err)

	if _, ok := result["total_results"]; ok {
		return GetResources(cliConnection, url, relationsDepth, follow, cache)
	}

	return GetResource(cliConnection, url, relationsDepth, follow, cache)
}

func GetResource(cliConnection plugin.CliConnection, url string, relationsDepth int, follow FollowDecision, cache map[string]interface{}) *models.ResourceModel {
	if cacheEntry, hit := cache[url]; hit {
		return cacheEntry.(*models.ResourceModel)
	}

	output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", url, "-X", "GET")
	FreakOut(err)

	resource := &models.ResourceModel{}

	err = json.Unmarshal([]byte(output[0]), &resource)
	FreakOut(err)

	cache[resource.Metadata["url"].(string)] = resource

	FollowRelation(cliConnection, resource, relationsDepth, follow, cache)

	return resource
}

func GetResources(cliConnection plugin.CliConnection, url string, relationsDepth int, follow FollowDecision, cache map[string]interface{}) *[]*models.ResourceModel {
	if cacheEntry, hit := cache[url]; hit {
		return cacheEntry.(*[]*models.ResourceModel)
	}

	nextURL := url
	allResources := make([]*models.ResourceModel, 0)

	for nextURL != "" {
		collectionModel := models.ResourceCollectionModel{}

		output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", nextURL, "-X", "GET")
		FreakOut(err)

		err = json.Unmarshal([]byte(output[0]), &collectionModel)
		FreakOut(err)

		for _, resource := range *collectionModel.Resources {
			if cacheEntry, hit := cache[resource.Metadata["url"].(string)]; hit {
				resource = cacheEntry.(*models.ResourceModel)
			} else {
				cache[resource.Metadata["url"].(string)] = resource
			}

			allResources = append(allResources, resource)
			FollowRelation(cliConnection, resource, relationsDepth, follow, cache)
		}

		nextURL = collectionModel.NextURL
	}

	cache[url] = &allResources

	return &allResources
}

func RemoveResourceRefs(resource *models.ResourceModel) {
	for childKey, childValue := range resource.Entity {
		_, isResourceModel := childValue.(*models.ResourceModel)
		_, isArrayResourceModel := childValue.(*[]*models.ResourceModel)

		if isResourceModel || isArrayResourceModel {
			delete(resource.Entity, childKey)
		}
	}
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
							RemoveResourceRefs(&childResourceCopy)
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

func RecreateLinkForEntity(resource *models.ResourceModel, cache map[string]interface{}) {
	for k, v := range resource.Entity {
		if strings.HasSuffix(k, models.UrlSuffix) {
			childURL := v.(string)
			childKey := strings.TrimSuffix(k, models.UrlSuffix)
			if cacheEntry, hit := cache[childURL]; hit {
				resource.Entity[childKey] = cacheEntry
			} else {
				if childEntity, hasEntity := resource.Entity[childKey]; hasEntity {
					childResource := TransformToResourceGeneric(childEntity, cache)
					cache[childURL] = childResource
					resource.Entity[childKey] = childResource
				}
			}
		}
	}
}

func TransformToResourceGeneric(r interface{}, cache map[string]interface{}) interface{} {
	switch r.(type) {
	case map[string]interface{}:
		return TransformToResource(r, cache)
	case []interface{}:
		return TransformToResources(r, cache)
	}

	panic("unknown resource type")
}

func TransformToResource(resource interface{}, cache map[string]interface{}) *models.ResourceModel {
	resourceModel := models.ResourceModel{}

	resourceEntitcache := resource.(map[string]interface{})

	resourceModel.Entity = resourceEntitcache["entity"].(map[string]interface{})
	resourceModel.Metadata = resourceEntitcache["metadata"].(map[string]interface{})

	RecreateLinkForEntity(&resourceModel, cache)

	return &resourceModel
}

func TransformToResources(resources interface{}, cache map[string]interface{}) *[]*models.ResourceModel {
	var result []*models.ResourceModel

	resourceArray := resources.([]interface{})

	for _, r := range resourceArray {
		resourceModel := TransformToResource(r, cache)
		result = append(result, resourceModel)
	}

	return &result
}

func GetOrgsResourcesRecurively(cliConnection plugin.CliConnection) (interface{}, error) {
	resourceUrlsWhitelistSlice := []interface{}{
		"organizations",
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

	cache := make(map[string]interface{})
	resources := GetResources(cliConnection, "/v2/organizations", 10, follow, cache)
	resources = BreakResourceLoops(resources)

	return resources, nil
}

func GetSharedDomains(cliConnection plugin.CliConnection) (interface{}, error) {

	follow := func(childKey string) bool {
		return false
	}

	cache := make(map[string]interface{})
	resources := GetResources(cliConnection, "/v2/shared_domains", 1, follow, cache)
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
