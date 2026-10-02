package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// RunGateServer embeds the UnimplementedRunGatesServer.
type RunGateServer struct {
	pb.UnimplementedRunGatesServer
	serviceCatalog *services.Catalog
}

// NewRunGateServer returns an instance of RunGateServer.
func NewRunGateServer(serviceCatalog *services.Catalog) *RunGateServer {
	return &RunGateServer{
		serviceCatalog: serviceCatalog,
	}
}

// ApproveRunGate records the caller's approve or reject decision on a run gate.
func (s *RunGateServer) ApproveRunGate(ctx context.Context, req *pb.ApproveRunGateRequest) (*pb.RunGate, error) {
	gateID, err := s.serviceCatalog.FetchModelID(ctx, req.GateId)
	if err != nil {
		return nil, err
	}

	// An unspecified decision maps to "unspecified", which the service rejects as invalid.
	gate, err := s.serviceCatalog.RunService.ApproveRunGate(ctx, &run.ApproveRunGateInput{
		GateID:   gateID,
		Decision: enumFromPB[models.RunGateDecision](req.Decision, "RUN_GATE_DECISION_"),
		Comment:  req.Comment,
	})
	if err != nil {
		return nil, err
	}

	return toPBRunGateWithApprovals(ctx, s.serviceCatalog, gate)
}

// OverrideRunGate clears a run gate without collecting its required approvals.
func (s *RunGateServer) OverrideRunGate(ctx context.Context, req *pb.OverrideRunGateRequest) (*pb.RunGate, error) {
	gateID, err := s.serviceCatalog.FetchModelID(ctx, req.GateId)
	if err != nil {
		return nil, err
	}

	gate, err := s.serviceCatalog.RunService.OverrideRunGate(ctx, gateID, req.Comment)
	if err != nil {
		return nil, err
	}

	return toPBRunGateWithApprovals(ctx, s.serviceCatalog, gate)
}

// toPBRunGateWithApprovals loads a gate's approvals so its per-rule approval counts can be reported.
func toPBRunGateWithApprovals(ctx context.Context, serviceCatalog *services.Catalog, gate *models.RunGate) (*pb.RunGate, error) {
	approvals, err := serviceCatalog.RunService.GetRunGateApprovalsByGateID(ctx, gate.Metadata.ID)
	if err != nil {
		return nil, err
	}

	return toPBRunGate(gate, approvals), nil
}

// toPBRunGate converts a run gate and its approvals to ProtoBuf form, reporting each rule's
// progress as the number of distinct principals that have approved it.
func toPBRunGate(gate *models.RunGate, approvals []models.RunGateApproval) *pb.RunGate {
	counts := gate.ApprovalCounts(approvals)
	pbGate := &pb.RunGate{
		Metadata:      toPBMetadata(&gate.Metadata, types.RunGateModelType),
		PolicyCheckId: models.RunNodeGID(gate.PolicyCheckID),
		Type:          enumToPB[pb.RunGateType](gate.Type, pb.RunGateType_value, "RUN_GATE_TYPE_"),
		Status:        enumToPB[pb.RunGateStatus](gate.Status, pb.RunGateStatus_value, "RUN_GATE_STATUS_"),
		ApprovalRules: make([]*pb.RunGateApprovalRule, 0, len(gate.ApprovalRules)),
		RunId:         gid.ToGlobalID(types.RunModelType, gate.RunID),
		OverriddenBy:  gate.OverriddenBy,
	}
	if gate.OverrideComment != nil {
		pbGate.OverrideComment = *gate.OverrideComment
	}
	for _, rule := range gate.ApprovalRules {
		pbGate.ApprovalRules = append(pbGate.ApprovalRules, &pb.RunGateApprovalRule{
			Name:              rule.Name,
			RequiredApprovals: int32(rule.RequiredApprovals),
			Approvals:         int32(counts[rule.Name]),
		})
	}
	return pbGate
}
