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
	if cacheEntry, hit := cache[url]; hit {
		return cacheEntry
	}

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

	log.Println("Retrinving resource", url)

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

	log.Println("Retrinving resources from", url)

	nextURL := url
	allResources := make([]*models.ResourceModel, 0)

	for nextURL != "" {
		collectionModel := models.ResourceCollectionModel{}

		output, err := cliConnection.CliCommandWithoutTerminalOutput("curl", nextURL, "-X", "GET")
		FreakOut(err)

		err = json.Unmarshal([]byte(output[0]), &collectionModel)
		FreakOut(err)

		log.Printf("Retrived %v/%v from %v", len(allResources)+len(*collectionModel.Resources), collectionModel.TotalResults, url)

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

func RecreateLinkForEntity(resource *models.ResourceModel, cache map[string]interface{}, follow FollowDecision) {
	for k, v := range resource.Entity {
		if strings.HasSuffix(k, models.UrlSuffix) {
			childURL, isValidUrlEntry := v.(string)
			if !isValidUrlEntry && childURL != "" {
				continue
			}

			childKey := strings.TrimSuffix(k, models.UrlSuffix)

			if !(follow == nil || follow(childKey)) {
				continue
			}

			if childEntity, hasEntity := resource.Entity[childKey]; hasEntity {
				childResource := TransformToResourceGeneric(childEntity, cache, follow)

				resource.Entity[childKey] = childResource

				if cacheEntry, hit := cache[childURL]; hit {
					if resourceCacheEntry, isSingleResource := cacheEntry.(*models.ResourceModel); isSingleResource {
						if resourceCacheEntry.Entity == nil {
							child := childResource.(*models.ResourceModel)
							resourceCacheEntry.Entity = child.Entity
						}
					}
				} else {
					cache[childURL] = childResource
				}
			} else {
				if cacheEntry, hit := cache[childURL]; hit {
					resource.Entity[childKey] = cacheEntry
				}
			}
		}
	}
}

func TransformToResourceGeneric(r interface{}, cache map[string]interface{}, follow FollowDecision) interface{} {
	switch r.(type) {
	case map[string]interface{}:
		return TransformToResource(r, cache, follow)
	case []interface{}:
		return TransformToResources(r, cache, follow)
	}

	panic("unknown resource type")
}

func TransformToResource(resource interface{}, cache map[string]interface{}, follow FollowDecision) *models.ResourceModel {
	resourceModel := models.ResourceModel{}

	resourceValue := resource.(map[string]interface{})

	resourceModel.Metadata = resourceValue["metadata"].(map[string]interface{})
	entity, hasEntity := resourceValue["entity"].(map[string]interface{})

	// Cache handling

	resourceUrl := resourceModel.Metadata["url"].(string)
	if cacheEntry, hit := cache[resourceUrl]; hit {
		cacheEntryValue := cacheEntry.(*models.ResourceModel)

		cacheEntryHasEntity := cacheEntryValue.Entity != nil
		if hasEntity && !cacheEntryHasEntity {
			cacheEntryValue.Entity = resourceModel.Entity
		}
		if !hasEntity && cacheEntryHasEntity {
			resourceModel.Entity = cacheEntryValue.Entity
		}
	} else {
		cache[resourceUrl] = &resourceModel
	}

	if hasEntity {
		resourceModel.Entity = entity
		RecreateLinkForEntity(&resourceModel, cache, follow)
	}

	return &resourceModel
}

func TransformToResources(resources interface{}, cache map[string]interface{}, follow FollowDecision) *[]*models.ResourceModel {
	var result []*models.ResourceModel

	resourceArray := resources.([]interface{})

	for _, r := range resourceArray {
		resourceModel := TransformToResource(r, cache, follow)
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

func ReadBackupJSON(jsonBytes []byte) (*models.BackupModel, error) {
	backupModel := models.BackupModel{}
	err := json.Unmarshal([]byte(jsonBytes), &backupModel)
	if err != nil {
		return nil, err
	}

	return &backupModel, nil
}
