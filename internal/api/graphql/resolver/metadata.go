package resolver

import (
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"

	graphql "github.com/graph-gophers/graphql-go"
)

// MetadataInput encapsulates the resource metadata input request
type MetadataInput struct {
	Version string
}

// MetadataResolver resolves the ResourceMetadata type
type MetadataResolver struct {
	metadata *models.ResourceMetadata
}

// Version resolver
func (r *MetadataResolver) Version() string {
	return strconv.Itoa(r.metadata.Version)
}

// CreatedAt resolver
func (r *MetadataResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: *r.metadata.CreationTimestamp}
}

// UpdatedAt resolver
func (r *MetadataResolver) UpdatedAt() graphql.Time {
	return graphql.Time{Time: *r.metadata.LastUpdatedTimestamp}
}

// TRN resolver
func (r *MetadataResolver) TRN() string {
	return r.metadata.TRN
}
