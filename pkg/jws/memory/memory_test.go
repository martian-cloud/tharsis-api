package memory

import (
	"context"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
)

const testKey = "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBeDNlOVpEb25mRXJ5ajlYaHczczFhNzZyaVY1UWN3bFFjbGRJZTVkdnM0Z09JMmJ6CnZPL1VvWC84ZzNJd0phY1FGVmxBWXZNOUlxR2NVM3Y5TzdNWkFxaHlVblF4ZUhVaWRZaDU2R2lZcFJZbEdmbEIKTTl2bmgyRVdwUTVsay90MUZZNGpCVVhWTlJORUw0RUhpZjlLNWFWUStqQk8vRUFVbkhPQ0w5U05MbmhGQlpOSQpBZEFRV2JHUC9WclhUTFBac0sxcDFwRzYxbGR5UGNGSG5UdEI4RDhkeFMxRHNlYzJGdHczQlg5eVpqOWM1VlRVCmsvSi9ZVThXQjJWcVVucmptU25rSEhDc04zNy96WkxlK1pTYkpmcWFVVWNhU3dYSTVzL2wyc1JlRFZCRzVSZmEKNEtXV2kzSm8zZjd1UXc1Q2xEN0RNTE9qNCsyaUJkMFdxS3RLa3dJREFRQUJBb0lCQUJ5THUxUHpxcUQwRE9DZQp5RTkwZTRHdTZOZWc0cnlEMGJwN1dVa3VzL0txZnZOOEVWZXhydWxwNnBaWktpREpWdGpuUy9xZE90dVE2MUZiClJqTmIvekZOTUFjeXo3MmdiYzBLT2dBVHIveTRQWmtJL2RiUkdzSmFGNmFKZCtvRXE0M3BLSVBocVpDTjhFMG8KRnc4ZHRQZEdnRjg5dUtBSy82Z2dNN3BxbmNYUGVXVStCMTJESzhHT05hR2VNeTAvL2ZIOUVJbmhjMFAvaHpzVQpOYjV0VHlSK3VYUzIxdFgvVm8yOWk1ZktqM24xd1ZLeWUveFNCL0l1elErd2t6bnowQTNPTHNwM3BreVowN0txCmt4VGpNdjBIRTdKV0pCUlNXQU13K0RlMkc5bVk1OVFra0k3MDdHUzhkYkNkNnA0NVgvNnROUmgxZ1h5V0FuNTUKK1FDSWt0RUNnWUVBL0lPZytXa2tzSWVSbDdxTmdLTURVODF5NXlsY1ZVMXpYekN4Qm5SUmNWalZQWDY2UW1ERwpGNm9HdzhJWTlkYnhuRno1MDhDZDNieXRnYnlLWFZEZHVBRmJsVmhmUEFZSGdxb2ZyL1MvQzF0ZTJiZFNIM3luCnFLaTUrU2FsN05TVlp4WEZYNVhRZTF6NmUrdkY1K3dZbHZ2YmhzN1FCNXJGV3pGam1TeEw3b3NDZ1lFQXlqaWwKMWtyZExpMjZXR1lTWm9sSkpzUFd5WkF4dkM4bWlCTlVUa3h6TTlCdnZ1K2RuL3pCZGtraEUxM2ZVSGJRVXVCTwpKK3JVeGpacnVETUNmcEVGV3U3WDN6aC9KTEE0ck1FOEtlM3RFT3NCZlFqeFpJak85RlpuNzF3alhncU5zSHFkCmNSL1VZdzhBVlo3WTRxSkxTZTZwWStvb0gxcWMwd0VCM3M1WjNSa0NnWUFUS2pKWGU2RnJQSDdTVXpoV0lRa3AKblJneGJ6UXd6VFlLYlhtaUVjWDBvbGRjMlhkdGZrYkttUDcyY0k3UWFjWGdwalhYMm5DZzJhY1poNlBBdlZoMgpsTFBNdSs1T2NlNnovaEYxbTArOG51eXQvWG5nSmVmYnB5S21SRWFubU9MVjloTUsxQ1lFbHVQejc4c1BkUjNRCnA4RGJsR3E1aVFlZGlqd1M3Z2U5VFFLQmdRQzBkdlZZVmhzbytOb1J4WldxTjk0MllCTXdjaVJRWW11cWlFVjIKVXJlRWlBRVJqbGxHeTVRQzhhUTQyazlPU1pvRU8vVERuc2dmMDdVeC95Tzh1OExoc2NDb2pveTR1eUw0Mk1ITwpVV3g4NnB5N2J0MnQ4SUZ3Z0NQazhuOEZqN2wrd3czNlJJT1BtM1dhODFWdWU1TmcrVUhsenJQYnQwdTQ2bTgyCjNVUkpFUUtCZ0JhblltZXMrMy8yeGpHQ29JWkE0S2t0YTN1dkc2UXlHQm92Mm5qZmFDZ2dNL1JKT3ZhdlhNb1QKZGNpSy9TdXpxd3V0ZW1oOEU5ZldGeGZTaytvY2RYd2dPeWRHQnNxeHUraGxGOXc0d29xTUVJdTJJcytxblp1eApNQ3FHZ3ZYdVlRdExyOUx2dG5RRDRFU1cxdXJsNTdtV3IwN0krdy9PQTl5OFh3NDA4T0p2Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="

func TestNewWithSigningKey(t *testing.T) {
	jwsProvider, err := New(map[string]string{
		"signing_key_b64": testKey,
	})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	assert.NotNil(t, jwsProvider.privKey)
}

func TestNewWithoutSigningKey(t *testing.T) {
	jwsProvider, err := New(map[string]string{})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	// Key should be automatically generated
	assert.NotNil(t, jwsProvider.privKey)
}

func TestSign(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pluginData := map[string]string{"signing_key_b64": testKey}

	jwsProvider, err := New(pluginData)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	token := jwt.New()
	err = token.Set(jwt.SubjectKey, "123")
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	signedToken, err := jwsProvider.Sign(ctx, payload)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	pubKey, err := jwk.FromRaw(jwsProvider.privKey.PublicKey)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	err = jwk.AssignKeyID(pubKey)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	_, err = jws.Verify(signedToken, jws.WithKey(jwa.RS256, pubKey))
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
}

func TestVerify(t *testing.T) {
	// Test cases
	tests := []struct {
		name      string
		privKey   string
		expectErr string
	}{
		{
			name:    "Valid Signature",
			privKey: testKey,
		},
		{
			name:      "Invalid signature",
			privKey:   "LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBNTBFYzE0b1ZKUFlIdDg2dzcrblhsdHlRNXQ5dUo3M2piNHZnUndVVlNLUFYweWtlCmZqakpIRyt4Q2c0U3FJWFlxajNMSnpOMjY2QVBzbUFUQ2JMQmI3a0FGME85V3hWYjV2bXdjUHh1RW44TXZzNjMKQWR6dFlXcUpSSUVTdndBU1c0eUlaSU9LeGNUS0ZpNjZJV2RaeUh4eWV4SkNMT0Y3eDU3N2w1ME82RzhQdzFSOQoyWFZkUGIrQmNQSDBHN0x4a0xEVVNDOENKYkVQVGZXRHJoaXVSRkh5UEdsTGdLY010R0VaeldaQjdPa2FneHVqCnBCTWl5Qy9PTC8rQklDeDV3Wm1UbTROTWZSbDBoMUhjaWlGTkh1TWFmRTFQdVF5MUp6MVZQcmVNOWFLYkRMa0UKUUZMWmN2MWp0K1k5UzUrdW9IVDBwUGRpRkorUDh6RjIxOElXUFFJREFRQUJBb0lCQVFDK1MxVlpoRFg4RVR5dQpvelgwWmovUzA3T2xXQXlFUlh5bktMb29sdVU1dmgvUlFFL29YQUFhRjByZTFFL0VQMGZZWnpzS0NnNTh2RnpPClVzSzN3MUhzQnBjdGpiOSsrU2VEL01tVllCT1B3dDZSRVBlRlgvQXlyb0ZBVHRwSE05N25XTkJQcmZMKzVQM0sKblM3ckYrbkkwYWMrNFBwMWhpTGVWRlFDSWVySGxqUTNMelpBNG1ROHl3K3ZvQnovZWh6cHBjQW03OGZ1YnpJcgpIRnNOL0U1TWxNaEhtd2JBTkdPV2NnR2hWeGljemV0eUlTaHYzeVMwWERrenErcDhwM2NqVVBOWWZXQ1ZvNGIwCnZLS2dBT0ZlSmloTlZBN3doaDQrTEI3STRWcWM2czljMkRvNWp4Z0QzT0EzQkRJc1JJT3N3MEgxSHFZY0F3Y3oKdlNNNEtlTmhBb0dCQVBTd1J3SWEzLzJvcmJ4bWdYWUt3d2tTQllwMnVsYk83Slp2bDBEUHVGUjR2bkFGVXZzWgp5Y1hDeWd2aGl6NWVQNlIwQ205WmN5QVA4REZOQnVNbXUzejF0cTVLbll6cndLMEM4SmhyK2wrR0xQK2JTT1A4Cm5TbmgrdXVFV2ErWHZQazBrazZMZlY1NkdYTnczVVpWVEtOVjRlcWc4SHFnMytLUExTdVRMMDVGQW9HQkFQSHgKMmFuS0xDekhleDgzUjhaTmpRTXdMeUNTdVlVb2JudUJxWEtucC94eVJCN3J0Q0ZyZWs3clg1bUs2eS9sQWMwagpJNnJXZHJDdUUzZk52Zm9Uem1zbndrekNtVFpkbStFL3pwWmRueDYwTkJCai9WS2xRRTJjVVQ0bWxBVk1ENEk5CmxqQUYyODlqVlIyaXFnQWJnUlhhSG1Ja1EvM3ZlbW0vVHlLWUk0T1pBb0dCQUpBejN2VTNuM0FmVzV4ZmtNWmYKVzBmYUxoZkhGdFFZQ01nenBhRVZpZDJHZHowUGRqTHpwTHorcWhKTWJzSm55dndCUXpFU04wM2E5c1FuVC9ySQpsYy8wQXlBK2F4RmswdDFqa1NWUzVYQXNaQktUa21hQ05xRTdRNlJQRUlmeVZmVkw3VG1LN1d5amFxSmxEcExuCnJFM0tUR1Q1U2lBSzlVYlErRjdvMUFVOUFvR0FHZlhSWFI2TVR1RzRuRWphTXJUdmhJQVBEbmV2NEZIT1NRSkkKcER6SkVaVlJLZUF3bThWa2drTlBKcko1T2RKZ3R4b21JWmFSZGJPMzh3cm9iNFRnaVM3aThrbVBGdjVFVTQ3OQpJN1UzOVp0d1dySGY1SlpHcUEwMXltMXBSSWc4d2NUSjhLMHdRTGh2MFpZNmwzaGNDWFExL21IVnlkR0FXUWhsCi9WaEZ0MEVDZ1lFQTdtK0RsR1YxZnRDUGd2V1ZEV3NjQk5DdWhET0dRaGhCdkdwWEw2ZXZhamQrSm9WcEFFM3kKbEJOZ2FKeW1lQzVBVlgrdWZKNmUxMFhTSkFPYUdyd2UyWHdmV1NuNER2OWpWWGlCWVcrUlA1OUxMc0Y2ZXE5RQpvNE1Wbks0THRadER6cWFwRm1ZbGF2dC9PS0k1emNjL2liYjlpa2RBQkZZcFlXclBlQkMvR2dNPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=",
			expectErr: "could not verify message using any of the signatures or keys",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			pluginData := map[string]string{"signing_key_b64": testKey}

			jwsProvider, err := New(pluginData)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			token := jwt.New()
			err = token.Set(jwt.SubjectKey, "123")
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			payload, err := jwt.NewSerializer().Serialize(token)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			hdrs := jws.NewHeaders()
			err = hdrs.Set(jws.TypeKey, "JWT")
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			parsedKey, err := parseBase64Key(test.privKey)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			jwkKey, err := jwk.FromRaw(parsedKey)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			// Set kid to the key ID used for the test key
			err = jwkKey.Set(jwk.KeyIDKey, "reN8YblgAUqmbNr3eAo5uc0lyrCUXodIgcktDmp39ig")
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			signedToken, err := jws.Sign(payload, jws.WithKey(jwa.RS256, jwkKey, jws.WithProtectedHeaders(hdrs)))
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			err = jwsProvider.Verify(ctx, signedToken)
			if test.expectErr == "" {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, test.expectErr)
			}
		})
	}
}
