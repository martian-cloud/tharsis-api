package servers

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var maxQueryLimit int32 = 100

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
	return &pb.ResourceMetadata{
		CreatedAt: timestamppb.New(*metadata.CreationTimestamp),
		UpdatedAt: timestamppb.New(*metadata.LastUpdatedTimestamp),
		Version:   int64(metadata.Version),
		Id:        gid.ToGlobalID(idType, metadata.ID),
		Trn:       metadata.TRN,
	}
}
