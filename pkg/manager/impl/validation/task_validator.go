// Miscellaneous functions to validate that required proto and spec fields are non empty when required for execution.
package validation

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/flyteorg/flyteadmin/pkg/repositories"

	"github.com/flyteorg/flyteadmin/pkg/common"
	"github.com/flyteorg/flyteadmin/pkg/errors"
	"github.com/flyteorg/flyteadmin/pkg/manager/impl/shared"
	runtime "github.com/flyteorg/flyteadmin/pkg/runtime/interfaces"
	runtimeInterfaces "github.com/flyteorg/flyteadmin/pkg/runtime/interfaces"
	"github.com/flyteorg/flyteidl/gen/pb-go/flyteidl/admin"
	"github.com/flyteorg/flyteidl/gen/pb-go/flyteidl/core"
	"github.com/flyteorg/flytestdlib/logger"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/resource"
)

var whitelistedTaskErr = errors.NewFlyteAdminErrorf(codes.InvalidArgument, "task type must be whitelisted before use")

// This is called for a task with a non-nil container.
func validateContainer(task core.TaskTemplate, taskConfig runtime.TaskResourceConfiguration) error {
	if err := ValidateEmptyStringField(task.GetContainer().Image, shared.Image); err != nil {
		return err
	}

	if task.GetContainer().Resources == nil {
		return nil
	}
	if err := validateTaskResources(task.Id, taskConfig.GetLimits(), task.GetContainer().Resources.Requests,
		task.GetContainer().Resources.Limits); err != nil {
		logger.Debugf(context.Background(), "encountered errors validating task resources for [%+v]: %v",
			task.Id, err)
		return err
	}
	return nil
}

func validatedK8sPodSpec(task *core.TaskTemplate) error {
	if task.GetK8SPod().PodSpec == nil {
		return errors.NewFlyteAdminErrorf(codes.InvalidArgument, "K8sPod task type targets must specify a non-empty pod spec")
	}
	var podSpec v1.PodSpec
	jsonObj, err := json.Marshal(task.GetK8SPod().PodSpec)
	if err != nil {
		return errors.NewFlyteAdminErrorf(codes.InvalidArgument, "Failed to json marshal K8sPod task type target with err: %v", err)
	}
	if err = json.Unmarshal(jsonObj, &podSpec); err != nil {
		return errors.NewFlyteAdminErrorf(codes.InvalidArgument, "Failed to json unmarshal K8sPod task type target with err: %v", err)
	}
	for _, container := range podSpec.Containers {
		if errs := validation.IsDNS1123Label(container.Name); len(errs) > 0 {
			return errors.NewFlyteAdminErrorf(codes.InvalidArgument, "Invalid container name [%s], err: %v", container.Name, errs)
		}
	}
	return nil
}

func validateRuntimeMetadata(metadata core.RuntimeMetadata) error {
	if err := ValidateEmptyStringField(metadata.Version, shared.RuntimeVersion); err != nil {
		return err
	}
	return nil
}

func validateTaskTemplate(taskID core.Identifier, task core.TaskTemplate,
	taskConfig runtime.TaskResourceConfiguration, whitelistConfig runtime.WhitelistConfiguration) error {
	if err := ValidateEmptyStringField(task.Type, shared.Type); err != nil {
		return err
	}
	if err := validateTaskType(taskID, task.Type, whitelistConfig); err != nil {
		return err
	}
	if task.Metadata == nil {
		return shared.GetMissingArgumentError(shared.Metadata)
	}
	if task.Metadata.Runtime != nil {
		if err := validateRuntimeMetadata(*task.Metadata.Runtime); err != nil {
			return err
		}
	}
	if task.Interface == nil {
		// The actual interface proto has nothing to validate.
		return shared.GetMissingArgumentError(shared.TypedInterface)
	}
	if task.GetContainer() != nil {
		return validateContainer(task, taskConfig)
	}
	if task.GetK8SPod() != nil {
		return validatedK8sPodSpec(&task)
	}
	return nil
}

func ValidateTask(
	ctx context.Context, request admin.TaskCreateRequest, db repositories.RepositoryInterface,
	taskConfig runtime.TaskResourceConfiguration, whitelistConfig runtime.WhitelistConfiguration,
	applicationConfig runtime.ApplicationConfiguration) error {
	if err := ValidateIdentifier(request.Id, common.Task); err != nil {
		return err
	}
	if err := ValidateProjectAndDomain(ctx, db, applicationConfig, request.Id.Project, request.Id.Domain); err != nil {
		return err
	}
	if request.Spec == nil || request.Spec.Template == nil {
		return shared.GetMissingArgumentError(shared.Spec)
	}
	return validateTaskTemplate(*request.Id, *request.Spec.Template, taskConfig, whitelistConfig)
}

func taskResourceSetToMap(
	resourceSet runtimeInterfaces.TaskResourceSet) map[core.Resources_ResourceName]*resource.Quantity {
	resourceMap := make(map[core.Resources_ResourceName]*resource.Quantity)
	if resourceSet.CPU != "" {
		cpuQuantity := resource.MustParse(resourceSet.CPU)
		resourceMap[core.Resources_CPU] = &cpuQuantity
	}
	if resourceSet.Memory != "" {
		memoryQuantity := resource.MustParse(resourceSet.Memory)
		resourceMap[core.Resources_MEMORY] = &memoryQuantity
	}
	if resourceSet.GPU != "" {
		gpuQuantity := resource.MustParse(resourceSet.GPU)
		resourceMap[core.Resources_GPU] = &gpuQuantity
	}
	return resourceMap
}

func addResourceEntryToMap(
	identifier *core.Identifier, entry *core.Resources_ResourceEntry,
	resourceEntries *map[core.Resources_ResourceName]resource.Quantity) error {
	if _, ok := (*resourceEntries)[entry.Name]; ok {
		return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
			"can't specify %v limit for task [%+v] multiple times", entry.Name, identifier)
	}
	quantity, err := resource.ParseQuantity(entry.Value)
	if err != nil {
		return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
			"Invalid quantity %s for resource: %v for task [%+v]", entry.Value, entry.Name, identifier)
	}
	(*resourceEntries)[entry.Name] = quantity
	return nil
}

func isWholeNumber(quantity resource.Quantity) bool {
	// Assert k8s quantity is a whole number
	return quantity.MilliValue()%1000 == 0
}

func requestedResourcesToQuantity(
	identifier *core.Identifier, resources []*core.Resources_ResourceEntry) (
	map[core.Resources_ResourceName]resource.Quantity, error) {

	var requestedToQuantity = make(map[core.Resources_ResourceName]resource.Quantity)
	for _, limitEntry := range resources {
		switch limitEntry.Name {
		case core.Resources_CPU:
			fallthrough
		case core.Resources_MEMORY:
			err := addResourceEntryToMap(identifier, limitEntry, &requestedToQuantity)
			if err != nil {
				return nil, err
			}
		case core.Resources_GPU:
			err := addResourceEntryToMap(identifier, limitEntry, &requestedToQuantity)
			if err != nil {
				return nil, err
			}
			if !isWholeNumber(requestedToQuantity[core.Resources_GPU]) {
				return nil, errors.NewFlyteAdminErrorf(codes.InvalidArgument,
					"gpu for [%+v] must be a whole number, got: %s instead", identifier, limitEntry.Value)
			}
		default:
			continue
		}
	}
	return requestedToQuantity, nil
}

func validateTaskResources(
	identifier *core.Identifier, taskResourceLimits runtimeInterfaces.TaskResourceSet,
	requestedTaskResourceDefaults, requestedTaskResourceLimits []*core.Resources_ResourceEntry) error {
	requestedResourceDefaults, err := requestedResourcesToQuantity(identifier, requestedTaskResourceDefaults)
	if err != nil {
		return err
	}

	requestedResourceLimits, err := requestedResourcesToQuantity(identifier, requestedTaskResourceLimits)
	if err != nil {
		return err
	}

	platformTaskResourceLimits := taskResourceSetToMap(taskResourceLimits)

	for resourceName, defaultQuantity := range requestedResourceDefaults {
		switch resourceName {
		case core.Resources_CPU:
			fallthrough
		case core.Resources_MEMORY:
			limitQuantity, ok := requestedResourceLimits[resourceName]
			if ok && limitQuantity.Value() < defaultQuantity.Value() {
				// Only assert the requested limit is greater than than the requested default when the limit is actually set
				return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
					"Type %v for [%+v] cannot set default > limit", resourceName, identifier)
			}
			platformLimit, platformLimitOk := platformTaskResourceLimits[resourceName]
			if ok && platformLimitOk && limitQuantity.Value() > platformLimit.Value() {
				// Also check that the requested limit is less than the platform task limit.
				return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
					"Type %v for [%+v] cannot set limit > platform limit", resourceName, identifier)
			}
			if platformLimitOk && defaultQuantity.Value() > platformTaskResourceLimits[resourceName].Value() {
				// Also check that the requested limit is less than the platform task limit.
				return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
					"Type %v for [%+v] cannot set default > platform limit", resourceName, identifier)
			}
		case core.Resources_GPU:
			limitQuantity, ok := requestedResourceLimits[resourceName]
			if ok && defaultQuantity.Value() != limitQuantity.Value() {
				return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
					"For extended resource 'gpu' the default value must equal the limit value for task [%+v]",
					identifier)
			}
			platformLimit, platformLimitOk := platformTaskResourceLimits[resourceName]
			if platformLimitOk && defaultQuantity.Value() > platformLimit.Value() {
				return errors.NewFlyteAdminErrorf(codes.InvalidArgument,
					"Type %v for [%+v] cannot set default > platform limit", resourceName, identifier)
			}
		}
	}

	return nil
}

func validateTaskType(taskID core.Identifier, taskType string, whitelistConfig runtime.WhitelistConfiguration) error {
	taskTypeWhitelist := whitelistConfig.GetTaskTypeWhitelist()
	if taskTypeWhitelist == nil {
		return nil
	}
	scopes, ok := taskTypeWhitelist[taskType]
	if !ok || scopes == nil || len(scopes) == 0 {
		return nil
	}
	for _, scope := range scopes {
		if scope.Project == "" {
			// All projects whitelisted
			return nil
		} else if scope.Project != taskID.Project {
			continue
		}
		// We have a potential match! Verify that this task type is approved given the specifity of the whitelist.
		if scope.Domain == "" {
			// All domains for this project are whitelisted
			return nil
		} else if scope.Domain == taskID.Domain {
			return nil
		}

	}
	return whitelistedTaskErr
}
