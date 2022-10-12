package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
)

// MeResponseResolver resolves the me query result
type MeResponseResolver struct {
	result interface{}
}

// ToUser handles User union type
func (r *MeResponseResolver) ToUser() (*UserResolver, bool) {
	res, ok := r.result.(*UserResolver)
	return res, ok
}

// ToServiceAccount handles ServiceAccount union type
func (r *MeResponseResolver) ToServiceAccount() (*ServiceAccountResolver, bool) {
	res, ok := r.result.(*ServiceAccountResolver)
	return res, ok
}

func meQuery(ctx context.Context) (*MeResponseResolver, error) {
	var response *MeResponseResolver

	if err := auth.HandleCaller(
		ctx,
		func(_ context.Context, c *auth.UserCaller) error {
			response = &MeResponseResolver{result: &UserResolver{user: c.User}}
			return nil
		},
		func(ctx context.Context, c *auth.ServiceAccountCaller) error {
			serviceAccount, err := getSAService(ctx).GetServiceAccountByID(ctx, c.ServiceAccountID)
			if err != nil {
				return err
			}
			response = &MeResponseResolver{result: &ServiceAccountResolver{serviceAccount: serviceAccount}}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return response, nil
}
