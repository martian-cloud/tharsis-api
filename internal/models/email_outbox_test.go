package models

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

func TestEmailOutboxItem(t *testing.T) {
	outbox := &EmailOutboxItem{
		EmailType:             builder.AnnouncementEmailType,
		Subject:               "hello",
		PayloadObjectStoreKey: new("emails/x/payload.json"),
		Status:                EmailOutboxItemStatusPreparing,
	}
	outbox.Metadata.ID = "outbox-1"

	assert.Equal(t, "outbox-1", outbox.GetID())
	assert.Equal(t, types.EmailOutboxItemModelType, outbox.GetModelType())
	assert.NotEmpty(t, outbox.GetGlobalID())
	assert.NoError(t, outbox.Validate())

	// ResolveMetadata delegates to the shared metadata resolver for a known field.
	val, err := outbox.ResolveMetadata("id")
	require.NoError(t, err)
	require.NotNil(t, val)
	assert.Equal(t, "outbox-1", *val)
}

func TestEmailOutboxItemValidate(t *testing.T) {
	valid := func() *EmailOutboxItem {
		return &EmailOutboxItem{
			EmailType:             builder.AnnouncementEmailType,
			Subject:               "hello",
			PayloadObjectStoreKey: new("emails/x/payload.json"),
			Status:                EmailOutboxItemStatusPreparing,
		}
	}

	tests := []struct {
		name    string
		mutate  func(*EmailOutboxItem)
		wantErr bool
	}{
		{name: "valid", mutate: func(*EmailOutboxItem) {}},
		{name: "invalid email type", mutate: func(o *EmailOutboxItem) { o.EmailType = "bogus" }, wantErr: true},
		{name: "empty subject", mutate: func(o *EmailOutboxItem) { o.Subject = "" }, wantErr: true},
		{name: "subject too long", mutate: func(o *EmailOutboxItem) { o.Subject = strings.Repeat("a", MaxEmailSubjectLength+1) }, wantErr: true},
		{
			name: "inline payload only is valid",
			mutate: func(o *EmailOutboxItem) {
				o.PayloadObjectStoreKey = nil
				o.Payload = json.RawMessage(`{"m":"hi"}`)
			},
		},
		{
			name:    "neither payload nor key is invalid",
			mutate:  func(o *EmailOutboxItem) { o.PayloadObjectStoreKey = nil },
			wantErr: true,
		},
		{
			name: "both payload and key is invalid",
			mutate: func(o *EmailOutboxItem) {
				o.Payload = json.RawMessage(`{"m":"hi"}`)
			},
			wantErr: true,
		},
		{name: "invalid status", mutate: func(o *EmailOutboxItem) { o.Status = "bogus" }, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o := valid()
			test.mutate(o)
			err := o.Validate()
			if test.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
