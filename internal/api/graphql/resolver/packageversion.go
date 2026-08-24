package resolver

import (
	"context"
	"encoding/hex"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/packageregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"

	graphql "github.com/graph-gophers/graphql-go"
)

/* PackageVersion Query Resolvers */

// PackageVersionsConnectionQueryArgs is used to query a package version connection
type PackageVersionsConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search *string
}

// PackageVersionEdgeResolver resolves package version edges
type PackageVersionEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *PackageVersionEdgeResolver) Cursor() (string, error) {
	pkgVersion, ok := r.edge.Node.(models.PackageVersion)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&pkgVersion)
	return *cursor, err
}

// Node returns a package version node
func (r *PackageVersionEdgeResolver) Node() (*PackageVersionResolver, error) {
	pkgVersion, ok := r.edge.Node.(models.PackageVersion)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &PackageVersionResolver{pkgVersion: &pkgVersion}, nil
}

// PackageVersionConnectionResolver resolves a package version connection
type PackageVersionConnectionResolver struct {
	connection Connection
}

// NewPackageVersionConnectionResolver creates a new PackageVersionConnectionResolver
func NewPackageVersionConnectionResolver(ctx context.Context, input *packageregistry.GetPackageVersionsInput) (*PackageVersionConnectionResolver, error) {
	service := getServiceCatalog(ctx).PackageService

	result, err := service.GetPackageVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	versions := result.PackageVersions

	// Create edges
	edges := make([]Edge, len(versions))
	for i, pkgVersion := range versions {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: pkgVersion}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(versions) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&versions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&versions[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &PackageVersionConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *PackageVersionConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the connection page information
func (r *PackageVersionConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *PackageVersionConnectionResolver) Edges() *[]*PackageVersionEdgeResolver {
	resolvers := make([]*PackageVersionEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &PackageVersionEdgeResolver{edge: edge}
	}
	return &resolvers
}

// PackageVersionResolver resolves a package version resource
type PackageVersionResolver struct {
	pkgVersion *models.PackageVersion
}

// ID resolver
func (r *PackageVersionResolver) ID() graphql.ID {
	return graphql.ID(r.pkgVersion.GetGlobalID())
}

// Version resolver
func (r *PackageVersionResolver) Version() string {
	return r.pkgVersion.SemanticVersion
}

// SHASum resolver
func (r *PackageVersionResolver) SHASum() string {
	return r.pkgVersion.GetSHASumHex()
}

// Size resolver returns the byte size of the uploaded package. It is 0 when the size is unknown:
// before the upload completes, or for versions uploaded before the field existed.
func (r *PackageVersionResolver) Size() int32 {
	return int32(r.pkgVersion.Size)
}

// Status resolver
func (r *PackageVersionResolver) Status() string {
	return toGraphqlEnum(string(r.pkgVersion.Status))
}

// Error resolver
func (r *PackageVersionResolver) Error() *string {
	return r.pkgVersion.Error
}

// Latest resolver
func (r *PackageVersionResolver) Latest() bool {
	return r.pkgVersion.Latest
}

// CreatedBy resolver
func (r *PackageVersionResolver) CreatedBy() string {
	return r.pkgVersion.CreatedBy
}

// Metadata resolver
func (r *PackageVersionResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.pkgVersion.Metadata}
}

// Package resolver
func (r *PackageVersionResolver) Package(ctx context.Context) (*PackageResolver, error) {
	pkg, err := loadPackage(ctx, r.pkgVersion.PackageID)
	if err != nil {
		return nil, err
	}

	return &PackageResolver{pkg: pkg}, nil
}

/* PackageVersion Mutation Resolvers */

// PackageVersionMutationPayload is the response payload for a package version mutation
type PackageVersionMutationPayload struct {
	ClientMutationID *string
	PackageVersion   *models.PackageVersion
	Problems         []Problem
}

// PackageVersionMutationPayloadResolver resolves a PackageVersionMutationPayload
type PackageVersionMutationPayloadResolver struct {
	PackageVersionMutationPayload
}

// PackageVersion field resolver
func (r *PackageVersionMutationPayloadResolver) PackageVersion() *PackageVersionResolver {
	if r.PackageVersionMutationPayload.PackageVersion == nil {
		return nil
	}
	return &PackageVersionResolver{pkgVersion: r.PackageVersionMutationPayload.PackageVersion}
}

// CreatePackageVersionInput contains the input for creating a new package version
type CreatePackageVersionInput struct {
	ClientMutationID *string
	PackageID        string
	Version          string
	SHASum           string
}

// DeletePackageVersionInput contains the input for deleting a package version
type DeletePackageVersionInput struct {
	ClientMutationID *string
	ID               string
}

func handlePackageVersionMutationProblem(e error, clientMutationID *string) (*PackageVersionMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := PackageVersionMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &PackageVersionMutationPayloadResolver{PackageVersionMutationPayload: payload}, nil
}

func createPackageVersionMutation(ctx context.Context, input *CreatePackageVersionInput) (*PackageVersionMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	packageID, err := serviceCatalog.FetchModelID(ctx, input.PackageID)
	if err != nil {
		return nil, err
	}

	shaSum, err := hex.DecodeString(input.SHASum)
	if err != nil {
		return nil, err
	}

	createdPackageVersion, err := serviceCatalog.PackageService.CreatePackageVersion(ctx, &packageregistry.CreatePackageVersionInput{
		PackageID:       packageID,
		SemanticVersion: input.Version,
		SHASum:          shaSum,
	})
	if err != nil {
		return nil, err
	}

	payload := PackageVersionMutationPayload{ClientMutationID: input.ClientMutationID, PackageVersion: createdPackageVersion, Problems: []Problem{}}
	return &PackageVersionMutationPayloadResolver{PackageVersionMutationPayload: payload}, nil
}

func deletePackageVersionMutation(ctx context.Context, input *DeletePackageVersionInput) (*PackageVersionMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	packageVersionID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	packageVersion, err := serviceCatalog.PackageService.GetPackageVersionByID(ctx, packageVersionID)
	if err != nil {
		return nil, err
	}

	if err := serviceCatalog.PackageService.DeletePackageVersion(ctx, packageVersion); err != nil {
		return nil, err
	}

	payload := PackageVersionMutationPayload{ClientMutationID: input.ClientMutationID, PackageVersion: packageVersion, Problems: []Problem{}}
	return &PackageVersionMutationPayloadResolver{PackageVersionMutationPayload: payload}, nil
}
