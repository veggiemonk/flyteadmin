package interfaces

import (
	"context"

	"github.com/lyft/flyteidl/gen/pb-go/flyteidl/admin"
)

// Interface for managing projects (and domains).
type ProjectDomainInterface interface {
	UpdateProjectDomain(ctx context.Context, request admin.ProjectDomainAttributesUpdateRequest) (
		*admin.ProjectDomainAttributesUpdateResponse, error)
}
