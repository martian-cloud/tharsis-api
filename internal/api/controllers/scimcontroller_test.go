package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestSCIMController_RespondWithSCIMError(t *testing.T) {
	tests := []struct {
		err          error
		name         string
		expectStatus int
		// expectScimType is the SCIM error type (RFC 7644 section 3.12) the response must carry.
		expectScimType string
	}{
		{
			name:           "a uniqueness conflict is reported as a uniqueness error",
			err:            errors.New("user already exists", errors.WithErrorCode(errors.EConflict)),
			expectStatus:   http.StatusConflict,
			expectScimType: "uniqueness",
		},
		{
			// A concurrent modification means the resource changed since it was read, not that a value
			// is already taken. Labeling it "uniqueness" would tell the identity provider the wrong thing.
			name:           "an optimistic lock error is a conflict that is not a uniqueness error",
			err:            errors.New("resource version does not match specified version", errors.WithErrorCode(errors.EOptimisticLock)),
			expectStatus:   http.StatusConflict,
			expectScimType: "",
		},
		{
			name:           "other errors carry no SCIM error type",
			err:            errors.New("not found", errors.WithErrorCode(errors.ENotFound)),
			expectStatus:   http.StatusNotFound,
			expectScimType: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testLogger, _ := logger.NewForTest()
			controller := &scimController{respWriter: response.NewWriter(testLogger), logger: testLogger}

			recorder := httptest.NewRecorder()
			controller.respondWithSCIMError(context.Background(), recorder, test.err)

			assert.Equal(t, test.expectStatus, recorder.Code)

			var body ScimErrorResponse
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			assert.Equal(t, test.expectScimType, body.ScimType)
			// The SCIM status is the HTTP status as a string (the RFC requires a string).
			assert.Equal(t, strconv.Itoa(test.expectStatus), body.Status)
		})
	}
}

func TestToListResponse(t *testing.T) {
	tests := []struct {
		name             string
		startIndex       string
		count            string
		expectErrorCode  errors.CodeType
		expectResources  []interface{}
		expectStartIndex int
		total            int
	}{
		{
			name:             "no parameters returns the default page size from the start",
			total:            25,
			expectStartIndex: 1,
			expectResources:  []interface{}{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
		},
		{
			name:             "startIndex and count select a page",
			total:            5,
			startIndex:       "2",
			count:            "2",
			expectStartIndex: 2,
			expectResources:  []interface{}{1, 2},
		},
		{
			// Identity providers look a resource up with a filter before provisioning it.
			name:             "a lookup with no results returns an empty list",
			total:            0,
			startIndex:       "1",
			count:            "100",
			expectStartIndex: 1,
			expectResources:  []interface{}{},
		},
		{
			name:             "startIndex past the end returns an empty list",
			total:            5,
			startIndex:       "7",
			count:            "10",
			expectStartIndex: 7,
			expectResources:  []interface{}{},
		},
		{
			name:             "startIndex below 1 is interpreted as 1",
			total:            3,
			startIndex:       "-4",
			count:            "1",
			expectStartIndex: 1,
			expectResources:  []interface{}{0},
		},
		{
			name:             "a negative count is interpreted as 0",
			total:            3,
			startIndex:       "1",
			count:            "-1",
			expectStartIndex: 1,
			expectResources:  []interface{}{},
		},
		{
			name:             "startIndex alone uses the default page size",
			total:            25,
			startIndex:       "21",
			expectStartIndex: 21,
			expectResources:  []interface{}{20, 21, 22, 23, 24},
		},
		{
			name:             "count alone starts at the first result",
			total:            5,
			count:            "2",
			expectStartIndex: 1,
			expectResources:  []interface{}{0, 1},
		},
		{
			name:            "a startIndex that is not an integer is rejected",
			total:           5,
			startIndex:      "abc",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "a count that is not an integer is rejected",
			total:           5,
			count:           "1.5",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := make([]int, test.total)
			for i := range items {
				items[i] = i
			}

			resp, err := toListResponse(items, test.startIndex, test.count)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}
			require.NoError(t, err)

			assert.Equal(t, test.expectResources, resp.Resources)
			assert.Equal(t, test.total, resp.TotalResults)
			assert.Equal(t, test.expectStartIndex, resp.StartIndex)
			assert.Equal(t, len(test.expectResources), resp.ItemsPerPage)
		})
	}
}

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name         string
		filter       string
		expectFilter *Filter
		expectError  bool
	}{
		{
			name:         "a simple filter is parsed",
			filter:       `userName eq "john"`,
			expectFilter: &Filter{Name: "username", Op: "eq", Value: "john"},
		},
		{
			name:         "externalId values are lowercased",
			filter:       `externalId EQ "ABC-123"`,
			expectFilter: &Filter{Name: "externalid", Op: "eq", Value: "abc-123"},
		},
		{
			name:         "a logical operator inside the quoted value is part of the value",
			filter:       `userName eq "tom and jerry"`,
			expectFilter: &Filter{Name: "username", Op: "eq", Value: "tom and jerry"},
		},
		{
			name:         "an OR inside the quoted value is part of the value",
			filter:       `externalId eq "this OR that"`,
			expectFilter: &Filter{Name: "externalid", Op: "eq", Value: "this or that"},
		},
		{
			name:         "an escaped quote does not end the quoted value",
			filter:       `userName eq "a\" and \"b"`,
			expectFilter: &Filter{Name: "username", Op: "eq", Value: `a\ and \b`},
		},
		{
			name:        "a logical operator after an escaped quote is not supported",
			filter:      `userName eq "a\"" and userName eq "b"`,
			expectError: true,
		},
		{
			name:        "an and filter is not supported",
			filter:      `userName eq "a" and externalId eq "b"`,
			expectError: true,
		},
		{
			name:        "an or filter is not supported, even with an operator inside a value",
			filter:      `userName eq "a or b" OR userName eq "c"`,
			expectError: true,
		},
		{
			name:        "an incomplete filter is not supported",
			filter:      `userName eq`,
			expectError: true,
		},
		{
			name:   "an empty filter is no filter",
			filter: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter, err := parseFilter(test.filter)
			if test.expectError {
				assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
				assert.Nil(t, filter)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expectFilter, filter)
		})
	}
}

func TestSCIMController_ResourceIDIsUnescaped(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expectID string
	}{
		{
			name:     "an id with no escaping",
			path:     "/scim/Users/VV86MTIz",
			expectID: gid.FromGlobalID("VV86MTIz"),
		},
		{
			name:     "an escaped id is unescaped",
			path:     "/scim/Users/a%2Cb%2Fc",
			expectID: gid.FromGlobalID("a,b/c"),
		},
		{
			// The request path is already decoded when it needs no escaping, so it is not decoded twice.
			name:     "an escaped percent sign is unescaped once",
			path:     "/scim/Users/%2541",
			expectID: gid.FromGlobalID("%41"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := &scimUserLookupStub{}
			testLogger, _ := logger.NewForTest()
			// Stands in for the JWT middleware: the SCIM handlers run with an authenticated caller.
			withCaller := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r.WithContext(auth.WithCaller(r.Context(), &auth.SystemCaller{})))
				})
			}
			router := chi.NewRouter()
			NewSCIMController(testLogger, response.NewWriter(testLogger), withCaller, svc, svc, svc).RegisterRoutes(router)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, test.path, nil))

			assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			assert.Equal(t, test.expectID, svc.getUserID)
		})
	}
}

// Aliases give the embedded service interfaces distinct field names.
type (
	scimStubSCIMService = scim.Service
	scimStubUserService = user.Service
	scimStubTeamService = team.Service
)

// scimUserLookupStub records the ID GetUserByID is called with. The scim, user and team services have no
// mockery mocks, so it embeds the interfaces: a call to any other method panics.
type scimUserLookupStub struct {
	scimStubSCIMService
	scimStubUserService
	scimStubTeamService

	getUserID string
}

func (s *scimUserLookupStub) GetUserByID(_ context.Context, id string) (*models.User, error) {
	s.getUserID = id
	return &models.User{Metadata: models.ResourceMetadata{ID: "user-id"}}, nil
}

func FuzzSCIMParseFilter(f *testing.F) {
	f.Add(`userName eq "john"`)
	f.Add(`externalId eq "ABC-123"`)
	f.Add(`USERNAME EQ "john doe"`)
	f.Add(`userName eq john`)
	f.Add(`userName eq "a" and externalId eq "b"`)
	f.Add(`userName eq "tom and jerry"`)
	f.Add(`externalId eq "this or that"`)
	f.Add(`userName eq "a" OR userName eq "b and c"`)
	f.Add(`userName  eq "x"`)
	f.Add(`userName`)
	f.Add(`userName eq`)
	f.Add(`emails[type eq "work"]`)
	f.Add("")
	f.Add(` eq ""`)

	f.Fuzz(func(t *testing.T, filter string) {
		parsed, err := parseFilter(filter)
		if err != nil {
			assert.Nil(t, parsed)
			assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			return
		}
		if filter == "" {
			assert.Nil(t, parsed)
			return
		}
		require.NotNil(t, parsed)

		// Attribute and operator are case-insensitive, so they are normalized; quotes are stripped from the value.
		assert.Equal(t, strings.ToLower(parsed.Name), parsed.Name)
		assert.Equal(t, strings.ToLower(parsed.Op), parsed.Op)
		assert.NotContains(t, parsed.Value, `"`)

		// A filter that passes validation for an attribute names exactly that attribute with the eq operator.
		if parsed.validate(userFilterAttributes) == nil {
			assert.Equal(t, scimFilterOperatorEq, parsed.Op)
			assert.Contains(t, userFilterAttributes, parsed.Name)
		}
	})
}

func FuzzSCIMParseFilterRoundTrip(f *testing.F) {
	f.Add("userName", "eq", "john")
	f.Add("externalId", "EQ", "ABC-123")
	f.Add("username", "eq", "john doe")
	f.Add("userName", "ne", "")
	f.Add("displayName", "eq", "x y z")

	f.Fuzz(func(t *testing.T, attribute, operator, value string) {
		// Build a filter in the form an identity provider sends it: attr op "value".
		if attribute == "" || operator == "" || strings.IndexFunc(attribute+operator, unicode.IsSpace) >= 0 {
			t.Skip()
		}
		if strings.Contains(attribute+operator+value, `"`) {
			t.Skip()
		}
		filter := attribute + " " + operator + ` "` + value + `"`
		if op := strings.ToLower(operator); op == "and" || op == "or" {
			// A logical operator outside the quoted value is a compound filter, which is not supported.
			_, err := parseFilter(filter)
			assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
			return
		}

		parsed, err := parseFilter(filter)
		require.NoError(t, err)
		require.NotNil(t, parsed)

		assert.Equal(t, strings.ToLower(attribute), parsed.Name)
		assert.Equal(t, strings.ToLower(operator), parsed.Op)
		if parsed.Name == scimFilterAttributeExternalID {
			assert.Equal(t, strings.ToLower(value), parsed.Value)
		} else {
			assert.Equal(t, value, parsed.Value)
		}
	})
}

func FuzzSCIMToListResponse(f *testing.F) {
	f.Add(uint8(0), "", "")
	f.Add(uint8(0), "1", "100")
	f.Add(uint8(5), "1", "2")
	f.Add(uint8(5), "3", "2")
	f.Add(uint8(5), "5", "10")
	f.Add(uint8(5), "6", "10")
	f.Add(uint8(5), "0", "0")
	f.Add(uint8(5), "-3", "2")
	f.Add(uint8(5), "1", "-1")
	f.Add(uint8(5), "abc", "1")
	f.Add(uint8(5), "1", "1.5")
	f.Add(uint8(30), "", "")
	f.Add(uint8(30), "2", "")
	f.Add(uint8(3), "99999999999999999999", "1")
	f.Add(uint8(0), "1", "")
	f.Add(uint8(30), "", "5")
	f.Add(uint8(5), "", "-7")

	f.Fuzz(func(t *testing.T, total uint8, startIndex, count string) {
		items := make([]int, total)
		for i := range items {
			items[i] = i
		}

		// Each parameter is optional (RFC 7644 section 3.4.2.4): startIndex defaults to 1 and count to 20.
		wantStart, wantCount := 1, 20
		var startErr, countErr error
		if startIndex != "" {
			wantStart, startErr = strconv.Atoi(startIndex)
		}
		if count != "" {
			wantCount, countErr = strconv.Atoi(count)
		}

		resp, err := toListResponse(items, startIndex, count)
		if startErr != nil || countErr != nil {
			// Only a parameter that is not an integer is rejected, as the client's fault.
			require.Error(t, err)
			assert.Equal(t, errors.EInvalid, errors.ErrorCode(err), "unexpected error: %v", err)
			return
		}
		require.NoError(t, err)
		require.NotNil(t, resp)

		// A startIndex below 1 is interpreted as 1, and a negative count as 0.
		wantStart, wantCount = max(wantStart, 1), max(wantCount, 0)

		assert.Equal(t, []SCIMSchemaURI{ListSchemaURI}, resp.SchemaURIs)
		assert.Equal(t, len(items), resp.TotalResults)
		assert.Equal(t, wantStart, resp.StartIndex)
		assert.NotNil(t, resp.Resources)

		// The page holds up to count items starting at the 1-based startIndex; past the end it is empty.
		first := wantStart - 1
		want := min(wantCount, max(0, len(items)-first))
		require.Len(t, resp.Resources, want)
		assert.Equal(t, want, resp.ItemsPerPage)
		for i, r := range resp.Resources {
			assert.Equal(t, first+i, r)
		}
	})
}
