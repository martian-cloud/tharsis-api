package types

import "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

// CreateManagedIdentityInput contains the fields for creating a new managed identity
type CreateManagedIdentityInput struct {
	Type        models.ManagedIdentityType
	Name        string
	Description string
	GroupID     string
	Data        []byte
	AccessRules []struct {
		RunStage                 models.JobType
		AllowedUserIDs           []string
		AllowedServiceAccountIDs []string
		AllowedTeamIDs           []string
	}
}

// UpdateManagedIdentityInput contains the fields for updating a managed identity
type UpdateManagedIdentityInput struct {
	ID          string
	Description string
	Data        []byte
}
