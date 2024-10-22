// Package tharsisfederated package
package tharsisfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/chmike/domain"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
)

// InputData contains the input data fields specific to this managed identity type
type InputData struct {
	ServiceAccountPath string   `json:"serviceAccountPath"`
	Hosts              []string `json:"hosts"`
}

// Data contains the data fields specific to this managed identity type
type Data struct {
	Subject            string   `json:"subject"`
	ServiceAccountPath string   `json:"serviceAccountPath"`
	Hosts              []string `json:"hosts"`
}

// Delegate for the Tharsis OIDC Federated managed identity type
type Delegate struct {
	jwsProvider jws.Provider
	issuerURL   string
}

// New creates a new Delegate instance
func New(_ context.Context, jwsProvider jws.Provider, issuerURL string) (*Delegate, error) {
	return &Delegate{
		jwsProvider: jwsProvider,
		issuerURL:   issuerURL,
	}, nil
}

// CreateCredentials returns a signed JWT token for the managed identity
func (d *Delegate) CreateCredentials(ctx context.Context, identity *models.ManagedIdentity, job *models.Job) ([]byte, error) {
	federatedData, err := decodeData(identity.Data)
	if err != nil {
		return nil, nil
	}

	currentTimestamp := time.Now().Unix()

	token := jwt.New()

	maxJobDuration := time.Duration(job.MaxJobDuration) * time.Minute
	if err = token.Set(jwt.ExpirationKey, time.Now().Add(maxJobDuration).Unix()); err != nil {
		return nil, err
	}
	if err = token.Set(jwt.NotBeforeKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err = token.Set(jwt.IssuedAtKey, currentTimestamp); err != nil {
		return nil, err
	}
	if err = token.Set(jwt.IssuerKey, d.issuerURL); err != nil {
		return nil, err
	}
	if err = token.Set(jwt.AudienceKey, "tharsis"); err != nil {
		return nil, err
	}
	if err = token.Set(jwt.SubjectKey, federatedData.Subject); err != nil {
		return nil, err
	}
	if err = token.Set("tharsis_job_id", gid.ToGlobalID(gid.JobType, job.Metadata.ID)); err != nil {
		return nil, err
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		return nil, err
	}

	return d.jwsProvider.Sign(ctx, payload)
}

// SetManagedIdentityData updates the managed identity custom data payload
func (d *Delegate) SetManagedIdentityData(_ context.Context, managedIdentity *models.ManagedIdentity, input []byte) error {
	decodedData, err := base64.StdEncoding.DecodeString(string(input))
	if err != nil {
		return te.Wrap(err, "failed to decode managed identity data", te.WithErrorCode(te.EInvalid))
	}

	inputData := InputData{}
	if err = json.Unmarshal(decodedData, &inputData); err != nil {
		return te.Wrap(err, "invalid managed identity data", te.WithErrorCode(te.EInvalid))
	}

	if inputData.ServiceAccountPath == "" {
		return errors.New("service account path field is missing from payload")
	}

	var federatedData *Data

	if managedIdentity.Data == nil || len(managedIdentity.Data) == 0 {
		federatedData = &Data{
			Subject: gid.ToGlobalID(gid.ManagedIdentityType, managedIdentity.Metadata.ID),
		}
	} else {
		federatedData, err = decodeData(managedIdentity.Data)
		if err != nil {
			return err
		}
	}

	err = validateHosts(inputData.Hosts)
	if err != nil {
		return err
	}

	federatedData.ServiceAccountPath = inputData.ServiceAccountPath
	federatedData.Hosts = inputData.Hosts

	buffer, err := json.Marshal(federatedData)
	if err != nil {
		return err
	}

	managedIdentity.Data = []byte(base64.StdEncoding.EncodeToString(buffer))

	return nil
}

func validateHosts(hosts []string) error {
	messages := make([]string, 0)

	messages = append(messages, validateHostsUnique(hosts)...)

	for _, hostAndPort := range hosts {
		messages = append(messages, validateHostWithPort(hostAndPort)...)
	}

	if len(messages) > 0 {
		return fmt.Errorf("invalid hosts: %v", messages)
	}

	return nil
}

func validateHostsUnique(hosts []string) []string {
	uniqueHosts := make(map[string]struct{}, len(hosts))
	messages := make([]string, 0)

	for _, host := range hosts {
		key := strings.ToLower(host)

		if _, ok := uniqueHosts[key]; ok {
			messages = append(messages, fmt.Sprintf("'%s': has already been specified", host))
		} else {
			uniqueHosts[key] = struct{}{}
		}
	}

	return messages
}

func validateHostWithPort(hostWithPort string) []string {
	messages := make([]string, 0)

	host, rawPort, err := net.SplitHostPort(hostWithPort)

	if err != nil {
		host = hostWithPort
	}

	if host != hostWithPort && rawPort == "" {
		messages = append(messages, fmt.Sprintf("'%s': port expected", hostWithPort))
	}

	err = validatePort(rawPort)
	if err != nil {
		messages = append(messages, fmt.Sprintf("'%s': invalid port, %v", hostWithPort, err))
	}

	err = domain.Check(host)
	if err != nil {
		messages = append(messages, fmt.Sprintf("'%s': %v", host, err))
	}

	return messages
}

func validatePort(rawPort string) error {
	if rawPort == "" {
		return nil
	}

	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return fmt.Errorf("port must be a valid integer")
	}

	if port < 0 || port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}

	return nil
}

func decodeData(data []byte) (*Data, error) {
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	federatedData := Data{}
	if err := json.Unmarshal(decodedData, &federatedData); err != nil {
		return nil, err
	}

	return &federatedData, nil
}
