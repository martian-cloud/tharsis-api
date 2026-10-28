package servers

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// toPBOptionalActor is toPBActor for an optional actor, returning nil when there
// is no actor.
func toPBOptionalActor(actor *models.Actor) *pb.Actor {
	if actor == nil {
		return nil
	}
	return toPBActor(*actor)
}

// toPBActor converts an Actor model to its ProtoBuf equivalent, setting the oneof
// case that matches the actor's type. An actor without a type yields nil, which
// is how a resource created before typed actors were introduced is represented;
// its Subject is exposed through the separate created_by field instead. No
// database access is required to produce it.
func toPBActor(actor models.Actor) *pb.Actor {
	if !actor.HasType() {
		return nil
	}

	principalID := actor.GlobalID()

	var trnSnapshot *string
	if actor.TRN != "" {
		actorTRN := actor.TRN
		trnSnapshot = &actorTRN
	}

	switch actor.Type {
	case models.ActorTypeUser:
		return &pb.Actor{Actor: &pb.Actor_User{User: &pb.UserActor{PrincipalId: principalID}}}
	case models.ActorTypeServiceAccount:
		return &pb.Actor{Actor: &pb.Actor_ServiceAccount{ServiceAccount: &pb.ServiceAccountActor{
			PrincipalId: principalID,
			TrnSnapshot: trnSnapshot,
		}}}
	case models.ActorTypeJob:
		return &pb.Actor{Actor: &pb.Actor_Job{Job: &pb.JobActor{PrincipalId: principalID, TrnSnapshot: trnSnapshot}}}
	case models.ActorTypeVCSProvider:
		return &pb.Actor{Actor: &pb.Actor_VcsProvider{VcsProvider: &pb.VCSProviderActor{
			PrincipalId: principalID,
			TrnSnapshot: trnSnapshot,
		}}}
	case models.ActorTypeSystem:
		return &pb.Actor{Actor: &pb.Actor_System{System: &pb.SystemActor{}}}
	case models.ActorTypeSCIM:
		return &pb.Actor{Actor: &pb.Actor_Scim{Scim: &pb.SCIMActor{}}}
	default:
		// Only reachable if an ActorType is added to the model before the proto.
		// Clients treat a missing actor as one recorded before typed actors.
		return nil
	}
}
