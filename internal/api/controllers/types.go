package controllers

// ServiceAccountLoginOptions is used to login to a service account
type ServiceAccountLoginOptions struct {
	// ServiceAccount needs to be set to the full path of the service account
	ServiceAccountPath *string `jsonapi:"attr,service-account-path,omitempty"`
	// Token is set to the token being used to login with
	Token *string `jsonapi:"attr,token,omitempty"`
	// Type is a public field utilized by JSON:API to
	// set the resource type via the field tag.
	Type string `jsonapi:"primary,service-account-token"`
}

// ServiceAccountLoginResponse is returned after logging in to a service account
type ServiceAccountLoginResponse struct {
	ID    string `jsonapi:"primary,service-account-token"`
	Token string `jsonapi:"attr,token"`
}
