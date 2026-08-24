package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/packageregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Package Query Resolvers */

// PackageConnectionQueryArgs are used to query a package connection
type PackageConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search           *string
	IncludeInherited *bool
}

// PackageEdgeResolver resolves package edges
type PackageEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *PackageEdgeResolver) Cursor() (string, error) {
	pkg, ok := r.edge.Node.(models.Package)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&pkg)
	return *cursor, err
}

// Node returns a package node
func (r *PackageEdgeResolver) Node() (*PackageResolver, error) {
	pkg, ok := r.edge.Node.(models.Package)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &PackageResolver{pkg: &pkg}, nil
}

// PackageConnectionResolver resolves a package connection
type PackageConnectionResolver struct {
	connection Connection
}

// NewPackageConnectionResolver creates a new PackageConnectionResolver
func NewPackageConnectionResolver(ctx context.Context, input *packageregistry.GetPackagesInput) (*PackageConnectionResolver, error) {
	service := getServiceCatalog(ctx).PackageService

	result, err := service.GetPackages(ctx, input)
	if err != nil {
		return nil, err
	}

	return packageConnectionResolver(result)
}

// NewVisiblePackageConnectionResolver creates a PackageConnectionResolver over every package the group
// may reference. It is a separate constructor because the visible listing is its own service method.
func NewVisiblePackageConnectionResolver(ctx context.Context, input *packageregistry.GetVisiblePackagesInput) (*PackageConnectionResolver, error) {
	service := getServiceCatalog(ctx).PackageService

	result, err := service.GetVisiblePackages(ctx, input)
	if err != nil {
		return nil, err
	}

	return packageConnectionResolver(result)
}

// packageConnectionResolver assembles the edges and page info shared by every package listing.
func packageConnectionResolver(result *db.PackagesResult) (*PackageConnectionResolver, error) {
	packages := result.Packages

	// Create edges
	edges := make([]Edge, len(packages))
	for i, pkg := range packages {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: pkg}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(packages) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&packages[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&packages[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &PackageConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *PackageConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the connection page information
func (r *PackageConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *PackageConnectionResolver) Edges() *[]*PackageEdgeResolver {
	resolvers := make([]*PackageEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &PackageEdgeResolver{edge: edge}
	}
	return &resolvers
}

// PackageResolver resolves a package resource
type PackageResolver struct {
	pkg *models.Package
}

// ID resolver
func (r *PackageResolver) ID() graphql.ID {
	return graphql.ID(r.pkg.GetGlobalID())
}

// Name resolver
func (r *PackageResolver) Name() string {
	return r.pkg.Name
}

// Description resolver
func (r *PackageResolver) Description() string {
	if r.pkg.Description == nil {
		return ""
	}
	return *r.pkg.Description
}

// Kind resolver
func (r *PackageResolver) Kind() string {
	return toGraphqlEnum(string(r.pkg.Kind))
}

// Visibility resolver
func (r *PackageResolver) Visibility() string {
	return toGraphqlEnum(string(r.pkg.Visibility))
}

// AllowMutableVersions resolver
func (r *PackageResolver) AllowMutableVersions() bool {
	return r.pkg.AllowMutableVersions
}

// CreatedBy resolver
func (r *PackageResolver) CreatedBy() string {
	return r.pkg.CreatedBy
}

// GroupPath resolver
func (r *PackageResolver) GroupPath() string {
	return r.pkg.GetGroupPath()
}

// Metadata resolver
func (r *PackageResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.pkg.Metadata}
}

// Group resolver
func (r *PackageResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.pkg.GroupID)
	if err != nil {
		return nil, err
	}
	return &GroupResolver{group: group}, nil
}

// Versions resolver
func (r *PackageResolver) Versions(ctx context.Context, args *PackageVersionsConnectionQueryArgs) (*PackageVersionConnectionResolver, error) {
	input := &packageregistry.GetPackageVersionsInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		PackageID: r.pkg.Metadata.ID,
		Search:    args.Search,
	}

	if args.Sort != nil {
		sort := db.PackageVersionSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewPackageVersionConnectionResolver(ctx, input)
}

// LatestVersion resolver
func (r *PackageResolver) LatestVersion(ctx context.Context) (*PackageVersionResolver, error) {
	versionsResp, err := getServiceCatalog(ctx).PackageService.GetPackageVersions(ctx, &packageregistry.GetPackageVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		PackageID: r.pkg.Metadata.ID,
		Latest:    ptr.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	if len(versionsResp.PackageVersions) == 0 {
		return nil, nil
	}

	return &PackageVersionResolver{pkgVersion: &versionsResp.PackageVersions[0]}, nil
}

// packagesQuery returns a package connection for a global (cross-group) listing. Results are
// limited by the caller's access (see the package service); admins see all packages.
func packagesQuery(ctx context.Context, args *PackageConnectionQueryArgs) (*PackageConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &packageregistry.GetPackagesInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		Search: args.Search,
	}

	if args.Sort != nil {
		sort := db.PackageSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewPackageConnectionResolver(ctx, input)
}

/* Package Mutation Resolvers */

// PackageMutationPayload is the response payload for a package mutation
type PackageMutationPayload struct {
	ClientMutationID *string
	Package          *models.Package
	Problems         []Problem
}

// PackageMutationPayloadResolver resolves a PackageMutationPayload
type PackageMutationPayloadResolver struct {
	PackageMutationPayload
}

// Package field resolver
func (r *PackageMutationPayloadResolver) Package() *PackageResolver {
	if r.PackageMutationPayload.Package == nil {
		return nil
	}
	return &PackageResolver{pkg: r.PackageMutationPayload.Package}
}

// CreatePackageInput contains the input for creating a package
type CreatePackageInput struct {
	ClientMutationID     *string
	Description          *string
	AllowMutableVersions *bool
	GroupID              string
	Name                 string
	Kind                 string
	Visibility           string
}

// UpdatePackageInput contains the input for updating a package
type UpdatePackageInput struct {
	ClientMutationID     *string
	Description          *string
	Visibility           *string
	AllowMutableVersions *bool
	ID                   string
}

// DeletePackageInput contains the input for deleting a package
type DeletePackageInput struct {
	ClientMutationID *string
	Force            *bool
	ID               string
}

func handlePackageMutationProblem(e error, clientMutationID *string) (*PackageMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := PackageMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &PackageMutationPayloadResolver{PackageMutationPayload: payload}, nil
}

func createPackageMutation(ctx context.Context, input *CreatePackageInput) (*PackageMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	groupID, err := serviceCatalog.FetchModelID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	createOptions := &packageregistry.CreatePackageInput{
		Name:       input.Name,
		GroupID:    groupID,
		Kind:       models.PackageKind(fromGraphqlEnum(input.Kind)),
		Visibility: models.PackageVisibility(fromGraphqlEnum(input.Visibility)),
	}

	if input.Description != nil {
		createOptions.Description = input.Description
	}

	if input.AllowMutableVersions != nil {
		createOptions.AllowMutableVersions = *input.AllowMutableVersions
	}

	pkg, err := serviceCatalog.PackageService.CreatePackage(ctx, createOptions)
	if err != nil {
		return nil, err
	}

	payload := PackageMutationPayload{ClientMutationID: input.ClientMutationID, Package: pkg, Problems: []Problem{}}
	return &PackageMutationPayloadResolver{PackageMutationPayload: payload}, nil
}

func updatePackageMutation(ctx context.Context, input *UpdatePackageInput) (*PackageMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	packageID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	updateOptions := &packageregistry.UpdatePackageInput{
		ID:                   packageID,
		Description:          input.Description,
		AllowMutableVersions: input.AllowMutableVersions,
	}

	if input.Visibility != nil {
		visibility := models.PackageVisibility(fromGraphqlEnum(*input.Visibility))
		updateOptions.Visibility = &visibility
	}

	pkg, err := serviceCatalog.PackageService.UpdatePackage(ctx, updateOptions)
	if err != nil {
		return nil, err
	}

	payload := PackageMutationPayload{ClientMutationID: input.ClientMutationID, Package: pkg, Problems: []Problem{}}
	return &PackageMutationPayloadResolver{PackageMutationPayload: payload}, nil
}

func deletePackageMutation(ctx context.Context, input *DeletePackageInput) (*PackageMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	packageID, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	pkg, err := serviceCatalog.PackageService.GetPackageByID(ctx, packageID)
	if err != nil {
		return nil, err
	}

	// force is accepted for API compatibility but no longer needed: attached resources are removed
	// by the database ON DELETE CASCADE.
	if err := serviceCatalog.PackageService.DeletePackage(ctx, pkg); err != nil {
		return nil, err
	}

	payload := PackageMutationPayload{ClientMutationID: input.ClientMutationID, Package: pkg, Problems: []Problem{}}
	return &PackageMutationPayloadResolver{PackageMutationPayload: payload}, nil
}

/* Package loader */

const packageLoaderKey = "package"

// RegisterPackageLoader registers a package loader function
func RegisterPackageLoader(collection *loader.Collection) {
	collection.Register(packageLoaderKey, packageBatchFunc)
}

func loadPackage(ctx context.Context, id string) (*models.Package, error) {
	ldr, err := loader.Extract(ctx, packageLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	pkg, ok := data.(models.Package)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &pkg, nil
}

func packageBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	packages, err := getServiceCatalog(ctx).PackageService.GetPackagesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range packages {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

/* PackageVersion loader */

const packageVersionLoaderKey = "packageVersion"

// RegisterPackageVersionLoader registers a package version loader function
func RegisterPackageVersionLoader(collection *loader.Collection) {
	collection.Register(packageVersionLoaderKey, packageVersionBatchFunc)
}

func loadPackageVersion(ctx context.Context, id string) (*models.PackageVersion, error) {
	ldr, err := loader.Extract(ctx, packageVersionLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	packageVersion, ok := data.(models.PackageVersion)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &packageVersion, nil
}

func packageVersionBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	packageVersions, err := getServiceCatalog(ctx).PackageService.GetPackageVersionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range packageVersions {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
