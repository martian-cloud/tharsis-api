package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// SCIMSchemaURI defines the SchemaURI used by SCIM resources.
// See: https://www.rfc-editor.org/rfc/rfc7644#section-8.2.
type SCIMSchemaURI string

// SCIMResourceType represents a SCIM resource type.
type SCIMResourceType string

// SCIMMetadata is the metadata for SCIM resources.
type SCIMMetadata struct {
	CreatedAt    *time.Time       `json:"created,omitempty"`
	LastModified *time.Time       `json:"lastModified,omitempty"`
	ResourceType SCIMResourceType `json:"resourceType"`
}

// ScimErrorResponse is the SCIM specific error response.
type ScimErrorResponse struct {
	Detail     string          `json:"detail"`
	Status     string          `json:"status"`
	ScimType   string          `json:"scimType,omitempty"`
	SchemaURIs []SCIMSchemaURI `json:"schemas"`
}

// SCIMEmail represents a SCIM user email.
type SCIMEmail struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

// SCIMUser represents a SCIM user resource.
type SCIMUser struct {
	SchemaURIs []SCIMSchemaURI `json:"schemas"`
	ID         string          `json:"id"`
	ExternalID string          `json:"externalId"`
	Metadata   SCIMMetadata    `json:"meta"`
	Username   string          `json:"userName"`
	Emails     []SCIMEmail     `json:"emails"`
	Active     bool            `json:"active"`
}

// SCIMGroup represents a SCIM group resource.
type SCIMGroup struct {
	Metadata    SCIMMetadata    `json:"meta"`
	ID          string          `json:"id"`
	ExternalID  string          `json:"externalId"`
	DisplayName string          `json:"displayName"`
	SchemaURIs  []SCIMSchemaURI `json:"schemas"`
}

// SCIMListResponse is a SCIM list response.
type SCIMListResponse struct {
	SchemaURIs   []SCIMSchemaURI `json:"schemas"`
	Resources    []interface{}   `json:"Resources"`
	TotalResults int             `json:"totalResults"`
	StartIndex   int             `json:"startIndex"`
	ItemsPerPage int             `json:"itemsPerPage"`
}

// CreateSCIMUserRequest represents a SCIM create user request.
type CreateSCIMUserRequest struct {
	ExternalID string          `json:"externalId"`
	Emails     []SCIMEmail     `json:"emails"`
	Schemas    []SCIMSchemaURI `json:"schemas"`
	Active     bool            `json:"active"`
}

// CreateSCIMGroupRequest represents a SCIM create group (Team) request.
type CreateSCIMGroupRequest struct {
	DisplayName string          `json:"displayName"`
	ExternalID  string          `json:"externalId"`
	Schemas     []SCIMSchemaURI `json:"schemas"`
}

// SCIMOperation represents a SCIM PATCH request operation.
type SCIMOperation struct {
	Value interface{} `json:"value"`
	OP    string      `json:"op"`
	Path  string      `json:"path"`
}

// SCIMUpdateRequest represents a SCIM update request.
type SCIMUpdateRequest struct {
	Schemas    []SCIMSchemaURI `json:"schemas"`
	Operations []SCIMOperation `json:"operations"`
}

// SCIMSchemaURI constants are used to indicate the schema type.
// SCIMResourceType constants are used to indicate the resource type being returned.
const (
	UserSchemaURI           SCIMSchemaURI = "urn:ietf:params:scim:schemas:core:2.0:User"
	UserEnterpriseSchemaURI SCIMSchemaURI = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	GroupSchemaURI          SCIMSchemaURI = "urn:ietf:params:scim:schemas:core:2.0:Group"
	ListSchemaURI           SCIMSchemaURI = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	UpdateSchemaURI         SCIMSchemaURI = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	ErrorSchemaURI          SCIMSchemaURI = "urn:ietf:params:scim:api:messages:2.0:Error"

	UserResourceType  SCIMResourceType = "User"
	GroupResourceType SCIMResourceType = "Group"
)

const (
	scimFilterAttributeExternalID = "externalid"
	scimFilterAttributeUsername   = "username"

	scimFilterOperatorEq = "eq"
)

var (
	// errUnsupportedFilter is an error used to indicate an invalid filter.
	errUnsupportedFilter = errors.New(
		"supplied filter is invalid or not supported",
		errors.WithErrorCode(errors.EInvalid),
	)

	// Allowed filter attributes per resource type
	userFilterAttributes  = []string{scimFilterAttributeExternalID, scimFilterAttributeUsername}
	groupFilterAttributes = []string{scimFilterAttributeExternalID}
)

type scimController struct {
	respWriter        response.Writer
	jwtAuthMiddleware middleware.Handler
	logger            logger.Logger
	userService       user.Service
	teamService       team.Service
	scimService       scim.Service
}

// NewSCIMController creates an instance of scimController
func NewSCIMController(
	logger logger.Logger,
	respWriter response.Writer,
	jwtAuthMiddleware middleware.Handler,
	userService user.Service,
	teamService team.Service,
	scimService scim.Service,
) Controller {
	return &scimController{
		respWriter,
		jwtAuthMiddleware,
		logger,
		userService,
		teamService,
		scimService,
	}
}

func (c *scimController) RegisterRoutes(router chi.Router) {
	// Require JTW auth.
	router.Use(c.jwtAuthMiddleware)

	// GET.
	router.Get("/scim/Users/{id}", c.GetUser)
	router.Get("/scim/Users", c.GetUsers)
	router.Get("/scim/Groups/{id}", c.GetGroup)
	router.Get("/scim/Groups", c.GetGroups)

	// POST.
	router.Post("/scim/Users", c.CreateUser)
	router.Post("/scim/Groups", c.CreateGroup)

	// PATCH.
	router.Patch("/scim/Users/{id}", c.UpdateUser)
	router.Patch("/scim/Groups/{id}", c.UpdateGroup)

	// DELETE.
	router.Delete("/scim/Users/{id}", c.DeleteUser)
	router.Delete("/scim/Groups/{id}", c.DeleteGroup)
}

/* User CRUD */

func (c *scimController) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := scimResourceID(r)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	user, err := c.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, TharsisUserToSCIMUser(user), http.StatusOK)
}

func (c *scimController) GetUsers(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	startIndex := r.URL.Query().Get("startIndex")
	count := r.URL.Query().Get("count")

	parsedFilter, err := parseFilter(filter)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	input := &scim.GetSCIMUsersInput{}
	if parsedFilter != nil {
		if err = parsedFilter.validate(userFilterAttributes); err != nil {
			c.respondWithSCIMError(r.Context(), w, err)
			return
		}

		if parsedFilter.Name == scimFilterAttributeExternalID {
			input.SCIMExternalID = &parsedFilter.Value
		} else if parsedFilter.Name == scimFilterAttributeUsername {
			input.Username = &parsedFilter.Value
		}
	}

	users, err := c.scimService.GetSCIMUsers(r.Context(), input)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	// Convert all users to SCIM equivalents.
	var scimUsers []SCIMUser
	for _, user := range users {
		aUser := user
		scimUsers = append(scimUsers, *TharsisUserToSCIMUser(&aUser))
	}

	response, err := toListResponse(scimUsers, startIndex, count)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, response, http.StatusOK)
}

func (c *scimController) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateSCIMUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.respondWithSCIMError(r.Context(), w, errors.Wrap(err, "failed to parse request body", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	// Find the primary email.
	var primaryEmail string
	for _, email := range req.Emails {
		if email.Primary {
			primaryEmail = email.Value
			break
		}
	}

	input := &scim.CreateSCIMUserInput{
		Email:          primaryEmail,
		SCIMExternalID: req.ExternalID,
		Active:         req.Active,
	}

	user, err := c.scimService.CreateSCIMUser(r.Context(), input)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, TharsisUserToSCIMUser(user), http.StatusCreated)
}

func (c *scimController) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := scimResourceID(r)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	var req SCIMUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.respondWithSCIMError(r.Context(), w, errors.Wrap(err, "failed to parse request body", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	// Prepare the input.
	input := &scim.UpdateResourceInput{
		ID:         userID,
		Operations: toSCIMOperations(req.Operations), // Avoids having to process the operations in the controller module.
	}

	updatedUser, err := c.scimService.UpdateSCIMUser(r.Context(), input)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, TharsisUserToSCIMUser(updatedUser), http.StatusOK)
}

func (c *scimController) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := scimResourceID(r)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	err = c.scimService.DeleteSCIMUser(r.Context(), &scim.DeleteSCIMResourceInput{ID: userID})
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, nil, http.StatusNoContent)
}

/* Teams (SCIM groups) CRUD */

func (c *scimController) GetGroup(w http.ResponseWriter, r *http.Request) {
	teamID, err := scimResourceID(r)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	team, err := c.teamService.GetTeamByID(r.Context(), teamID)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, TharsisTeamToSCIMGroup(team), http.StatusOK)
}

func (c *scimController) GetGroups(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	startIndex := r.URL.Query().Get("startIndex")
	count := r.URL.Query().Get("count")

	parsedFilter, err := parseFilter(filter)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	input := &scim.GetSCIMGroupsInput{}
	if parsedFilter != nil {
		if err = parsedFilter.validate(groupFilterAttributes); err != nil {
			c.respondWithSCIMError(r.Context(), w, err)
			return
		}
		input.SCIMExternalID = parsedFilter.Value
	}

	groups, err := c.scimService.GetSCIMGroups(r.Context(), input)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	// Convert all teams to SCIM equivalents.
	var scimGroups []SCIMGroup
	for _, team := range groups {
		aTeam := team
		scimGroups = append(scimGroups, *TharsisTeamToSCIMGroup(&aTeam))
	}

	response, err := toListResponse(scimGroups, startIndex, count)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, response, http.StatusOK)
}

func (c *scimController) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req CreateSCIMGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.respondWithSCIMError(r.Context(), w, errors.Wrap(err, "failed to parse request body", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	input := &scim.CreateSCIMGroupInput{
		Name:           req.DisplayName,
		SCIMExternalID: req.ExternalID,
	}

	team, err := c.scimService.CreateSCIMGroup(r.Context(), input)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, TharsisTeamToSCIMGroup(team), http.StatusCreated)
}

func (c *scimController) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	teamID, err := scimResourceID(r)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	var req SCIMUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.respondWithSCIMError(r.Context(), w, errors.Wrap(err, "failed to parse request body", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	input := &scim.UpdateResourceInput{
		ID:         teamID,
		Operations: toSCIMOperations(req.Operations),
	}

	// Returned model is not needed since SCIM never wants it.
	_, err = c.scimService.UpdateSCIMGroup(r.Context(), input)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	// Contrary to UpdateUsers, SCIM requires no content for group updates.
	c.respWriter.RespondWithJSON(r.Context(), w, nil, http.StatusNoContent)
}

func (c *scimController) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	teamID, err := scimResourceID(r)
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	err = c.scimService.DeleteSCIMGroup(r.Context(), &scim.DeleteSCIMResourceInput{ID: teamID})
	if err != nil {
		c.respondWithSCIMError(r.Context(), w, err)
		return
	}

	c.respWriter.RespondWithJSON(r.Context(), w, nil, http.StatusNoContent)
}

/* Conversions */

// TharsisUserToSCIMUser converts a Tharsis user to a SCIM user.
func TharsisUserToSCIMUser(user *models.User) *SCIMUser {
	return &SCIMUser{
		SchemaURIs: []SCIMSchemaURI{
			UserSchemaURI,
			UserEnterpriseSchemaURI,
		},
		ID:         user.GetGlobalID(),
		ExternalID: user.SCIMExternalID,
		Metadata: SCIMMetadata{
			CreatedAt:    user.Metadata.CreationTimestamp,
			LastModified: user.Metadata.LastUpdatedTimestamp,
			ResourceType: UserResourceType,
		},
		Username: user.Username,
		Emails: []SCIMEmail{
			{
				Type:    "work",
				Value:   user.Email,
				Primary: true,
			},
		},
		Active: user.Active,
	}
}

// TharsisTeamToSCIMGroup converts from Tharsis team to a SCIMGroup.
func TharsisTeamToSCIMGroup(team *models.Team) *SCIMGroup {
	return &SCIMGroup{
		Metadata: SCIMMetadata{
			ResourceType: GroupResourceType,
			CreatedAt:    team.Metadata.CreationTimestamp,
			LastModified: team.Metadata.LastUpdatedTimestamp,
		},
		SchemaURIs:  []SCIMSchemaURI{GroupSchemaURI},
		ID:          team.GetGlobalID(),
		DisplayName: team.Name,
		ExternalID:  team.SCIMExternalID,
	}
}

/* Custom responses */

// respondWithSCIMError responds to an http request with a SCIM error message.
func (c *scimController) respondWithSCIMError(ctx context.Context, w http.ResponseWriter, err error) {
	if !errors.IsContextCanceledError(err) &&
		errors.ErrorCode(err) != errors.EUnauthorized &&
		errors.ErrorCode(err) != errors.EForbidden &&
		errors.ErrorCode(err) != errors.ENotFound {
		c.logger.WithContextFields(ctx).Errorf("Unexpected error occurred: %s", err.Error())
	}

	code := response.ErrorCodeToStatusCode(errors.ErrorCode(err))
	scimErr := &ScimErrorResponse{
		Detail:     errors.ErrorMessage(err),
		SchemaURIs: []SCIMSchemaURI{ErrorSchemaURI},
		Status:     fmt.Sprintf("%d", code), // Must be a string.
	}

	// RFC 7644 section 3.12: a 409 for a create or update must identify the uniqueness conflict. An optimistic
	// lock error also maps to a 409, but it means the resource was modified concurrently, so it is not one.
	if errors.ErrorCode(err) == errors.EConflict {
		scimErr.ScimType = "uniqueness"
	}

	c.respWriter.RespondWithJSON(ctx, w, scimErr, code)
}

// toListResponse converts value to a SCIMListResponse with pagination,
// used primarily for returning a slice of SCIM resources.
func toListResponse(value interface{}, startIndex, count string) (*SCIMListResponse, error) {
	var (
		start        = 1 // Must be 1-based.
		itemsPerPage = 20
		valueSlice   = reflect.ValueOf(value)
		resources    = make([]interface{}, 0) // Avoid sending null field when count = 0.
		err          error
	)

	// Pagination per RFC 7644 section 3.4.2.4: each parameter is optional, a startIndex below 1 is
	// interpreted as 1, a negative count as 0, and a startIndex past the end returns an empty page.
	if startIndex != "" {
		start, err = strconv.Atoi(startIndex)
		if err != nil {
			return nil, errors.Wrap(err, "invalid startIndex", errors.WithErrorCode(errors.EInvalid))
		}
		start = max(start, 1)
	}

	if count != "" {
		itemsPerPage, err = strconv.Atoi(count)
		if err != nil {
			return nil, errors.Wrap(err, "invalid count", errors.WithErrorCode(errors.EInvalid))
		}
		itemsPerPage = max(itemsPerPage, 0)
	}

	// Append up to itemsPerPage values beginning from start.
	for i := start - 1; i < valueSlice.Len() && len(resources) < itemsPerPage; i++ {
		resources = append(resources, valueSlice.Index(i).Interface())
	}

	return &SCIMListResponse{
		SchemaURIs:   []SCIMSchemaURI{ListSchemaURI},
		Resources:    resources,
		TotalResults: valueSlice.Len(),
		StartIndex:   start,
		ItemsPerPage: len(resources), // The number of resources in this page (RFC 7644 section 3.4.2).
	}, nil
}

// Filter represents a SCIM filter with name, value, and operator.
type Filter struct {
	Name  string
	Value string
	Op    string
}

func (f *Filter) validate(allowedAttributes []string) error {
	if f.Op != scimFilterOperatorEq {
		return errUnsupportedFilter
	}
	for _, attr := range allowedAttributes {
		if f.Name == attr {
			return nil
		}
	}
	return errUnsupportedFilter
}

// parseFilter parses a simple request filter, such as, filter=userName Eq "john".
// Returns an error if the filter is not supported.
func parseFilter(filter string) (*Filter, error) {
	if filter == "" {
		return nil, nil
	}

	// Check for unsupported complex filters with logical operators. Only text outside quoted values is
	// checked, so a value such as "tom and jerry" is allowed.
	unquoted := strings.ToLower(withoutQuotedValues(filter))
	if strings.Contains(unquoted, " and ") || strings.Contains(unquoted, " or ") {
		return nil, errUnsupportedFilter
	}

	parts := strings.SplitAfterN(filter, " ", 3)
	if len(parts) < 3 {
		return nil, errUnsupportedFilter
	}

	// Lowercasing as these must be case-insensitive per RFC specifications.
	attribute := strings.ToLower(strings.TrimSpace(parts[0]))
	operator := strings.ToLower(strings.TrimSpace(parts[1]))
	value := strings.ReplaceAll(parts[2], "\"", "")

	// Only lowercase value for externalId attribute
	if attribute == scimFilterAttributeExternalID {
		value = strings.ToLower(value)
	}

	return &Filter{Name: attribute, Value: value, Op: operator}, nil
}

// withoutQuotedValues returns filter with the contents of its double-quoted values removed. A backslash
// inside a value escapes the next character, so an escaped quote does not end the value.
func withoutQuotedValues(filter string) string {
	var b strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range filter {
		switch {
		case inQuotes && escaped:
			escaped = false
		case inQuotes && r == '\\':
			escaped = true
		case r == '"':
			inQuotes = !inQuotes
			b.WriteRune(r)
		case !inQuotes:
			b.WriteRune(r)
		}
	}
	return b.String()
}

/* Helper functions */

// scimResourceID returns the global ID from the {id} URL parameter. chi matches the escaped path when the
// request path needed escaping, so the parameter is unescaped only in that case to avoid decoding twice.
func scimResourceID(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	if r.URL.RawPath != "" {
		var err error
		if id, err = url.PathUnescape(id); err != nil {
			return "", errors.Wrap(err, "invalid resource id", errors.WithErrorCode(errors.EInvalid))
		}
	}
	return gid.FromGlobalID(id), nil
}

// toSCIMOperations prepares the input for a SCIM operation.
func toSCIMOperations(operations []SCIMOperation) []scim.Operation {
	ops := make([]scim.Operation, len(operations))
	for x, operation := range operations {
		ops[x].OP = scim.OP(strings.ToLower(operation.OP))
		ops[x].Path = operation.Path
		ops[x].Value = operation.Value
	}

	return ops
}
