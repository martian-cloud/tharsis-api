package servers

import (
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var maxQueryLimit int32 = 100

// enumToPB maps a snake_case domain enum value to its prefixed protobuf enum value.
// Proto value names follow <PREFIX><UPPER_SNAKE> (the standard protobuf convention of
// prefixing values with the enum name), e.g. POLICY_CHECK_STATUS_SOFT_FAILED. This lets a
// new enum value flow through by convention instead of a per-value switch. Empty or unknown
// values fall back to 0 (the enum's *_UNSPECIFIED zero value).
func enumToPB[E ~int32, D ~string](v D, valueMap map[string]int32, prefix string) E {
	return E(valueMap[prefix+strings.ToUpper(string(v))])
}

// enumFromPB strips the protobuf prefix from a .String() value and lowercases it to
// recover the snake_case domain constant, e.g. "RUN_TASK_STAGE_NAME_QUEUED" with prefix
// "RUN_TASK_STAGE_NAME_" → "queued". Pass an empty prefix for enums that have none.
func enumFromPB[D ~string, E interface{ String() string }](v E, prefix string) D {
	return D(strings.ToLower(strings.TrimPrefix(v.String(), prefix)))
}

/* Conversions from ProtoBuf models */

// fromPBPaginationOptions converts ProtoBuf pagination options to API equivalent.
func fromPBPaginationOptions(opts *pb.PaginationOptions) (*pagination.Options, error) {
	if opts == nil {
		// Default to first 100 records if pagination options aren't being used.
		return &pagination.Options{First: &maxQueryLimit}, nil
	}

	if opts.First != nil && opts.Last != nil {
		return nil, errors.New("invalid args: only first or last may be used", errors.WithErrorCode(errors.EInvalid))
	}

	if opts.First == nil && opts.Last == nil {
		return nil, errors.New("invalid args: either first or last must be specified", errors.WithErrorCode(errors.EInvalid))
	}

	if opts.GetFirst() < 0 || opts.GetFirst() > maxQueryLimit {
		return nil, errors.New("invalid args: first must be between 0-%d", maxQueryLimit, errors.WithErrorCode(errors.EInvalid))
	}

	if opts.GetLast() < 0 || opts.GetLast() > maxQueryLimit {
		return nil, errors.New("invalid args: last must be between 0-%d", maxQueryLimit, errors.WithErrorCode(errors.EInvalid))
	}

	return &pagination.Options{
		Before: opts.Before,
		After:  opts.After,
		First:  opts.First,
		Last:   opts.Last,
	}, nil
}

/* Conversions to ProtoBuf models */

// toPBMetadata converts from ResourceMetadata model to ProtoBuf model.
func toPBMetadata(metadata *models.ResourceMetadata, idType types.ModelType) *pb.ResourceMetadata {
	return toPBMetadataWithGID(metadata, gid.ToGlobalID(idType, metadata.ID))
}

// toPBMetadataWithGID builds pb metadata from a pre-computed global ID. Used for run nodes (plan,
// apply, policy check) whose GID is a bare RunNode code with no ModelType.
func toPBMetadataWithGID(metadata *models.ResourceMetadata, globalID string) *pb.ResourceMetadata {
	return &pb.ResourceMetadata{
		CreatedAt: timestamppb.New(*metadata.CreationTimestamp),
		UpdatedAt: timestamppb.New(*metadata.LastUpdatedTimestamp),
		Version:   int64(metadata.Version),
		Id:        globalID,
		Trn:       metadata.TRN,
	}
}

// toPBPackageVersion converts from a PackageVersion model to its ProtoBuf representation. The
// metadata id is the global id used to reference the version elsewhere (e.g. the download endpoint).
func toPBPackageVersion(packageVersion *models.PackageVersion) *pb.PackageVersion {
	return &pb.PackageVersion{
		Metadata:  toPBMetadata(&packageVersion.Metadata, types.PackageVersionModelType),
		PackageId: packageVersion.PackageID,
		Version:   packageVersion.SemanticVersion,
		Status:    string(packageVersion.Status),
		ShaSum:    packageVersion.GetSHASumHex(),
	}
}
