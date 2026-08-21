package models

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

var _ Model = (*Package)(nil)

// PackageKind identifies the kind of artifact a package stores. Today only OPA policy bundles are
// supported; future kinds are added here.
type PackageKind string

// PackageKind constants
const (
	// PackageKindOPAPolicy is a package containing Open Policy Agent (OPA) policy bundles
	PackageKindOPAPolicy PackageKind = "opa_policy"
)

// PackageVisibility controls which namespaces a package is visible to
type PackageVisibility string

// PackageVisibility constants
const (
	// PackageVisibilityPrivate makes the package visible to the defining group and its subgroups
	PackageVisibilityPrivate PackageVisibility = "private"
	// PackageVisibilityRootGroup makes the package visible to all groups sharing the same root group
	PackageVisibilityRootGroup PackageVisibility = "root_group"
	// PackageVisibilityGlobal makes the package visible to all groups
	PackageVisibilityGlobal PackageVisibility = "global"
)

// Package is a group-owned, versioned artifact in the package registry, identified by its Kind.
type Package struct {
	Name        string
	Description *string
	GroupID     string
	RootGroupID string
	CreatedBy   string
	Kind        PackageKind
	Visibility  PackageVisibility
	Metadata    ResourceMetadata
	// AllowMutableVersions permits re-uploading the contents of an already-uploaded version in place
	// (without bumping the semantic version). When false, versions are immutable once uploaded.
	AllowMutableVersions bool
}

// GetID returns the Metadata ID.
func (p *Package) GetID() string {
	return p.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (p *Package) GetGlobalID() string {
	return gid.ToGlobalID(p.GetModelType(), p.Metadata.ID)
}

// GetModelType returns the model type.
func (p *Package) GetModelType() types.ModelType {
	return types.PackageModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (p *Package) ResolveMetadata(key string) (*string, error) {
	val, err := p.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "name":
			return &p.Name, nil
		case "group_path":
			path := p.GetGroupPath()
			return &path, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (p *Package) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(p.Name); err != nil {
		return err
	}

	// A package is always owned by a group and rooted at a top-level group; both back the visibility
	// checks and the FK columns, so neither may be empty. Mirrors Policy.Validate's group-owner check.
	if p.GroupID == "" {
		return errors.New("package must have a group owner", errors.WithErrorCode(errors.EInvalid))
	}
	if p.RootGroupID == "" {
		return errors.New("package must have a root group", errors.WithErrorCode(errors.EInvalid))
	}

	// Verify description satisfies constraints
	if p.Description != nil {
		if err := verifyValidDescription(*p.Description); err != nil {
			return err
		}
	}

	switch p.Kind {
	case PackageKindOPAPolicy:
	default:
		return errors.New("package kind %s is not supported", p.Kind, errors.WithErrorCode(errors.EInvalid))
	}

	switch p.Visibility {
	case PackageVisibilityPrivate, PackageVisibilityRootGroup, PackageVisibilityGlobal:
	default:
		return errors.New("package visibility %s is not supported", p.Visibility, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// GetResourcePath returns the resource path
func (p *Package) GetResourcePath() string {
	return trn.MustParseAny(p.Metadata.TRN).Path()
}

// GetGroupPath returns the group path
func (p *Package) GetGroupPath() string {
	return trn.MustParseAny(p.Metadata.TRN).ParentPath()
}

// GetRootGroupPath returns the root group path (the first segment) of the package's owning group.
func (p *Package) GetRootGroupPath() string {
	return strings.Split(p.GetGroupPath(), "/")[0]
}
