package secretsmanagement

import (
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
)

// ========= KeystoreResource =========

func TestKeystoreResource_flattenKeystore(t *testing.T) {
	r := &KeystoreResource{}

	t.Run("basic keystore is flattened", func(t *testing.T) {
		ks := &secretsmgmt.KeystoreResponse{
			Name: "my-keystore",
			Type: "PEM",
			Meta: secretsmgmt.SecretGroupMeta{ID: "ks-uuid-1"},
		}
		data := &KeystoreResourceModel{}
		r.flattenKeystore(ks, data, "org-1", "env-2", "sg-3")

		if data.ID.ValueString() != "ks-uuid-1" {
			t.Errorf("ID = %q, want ks-uuid-1", data.ID.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", data.EnvironmentID.ValueString())
		}
		if data.SecretGroupID.ValueString() != "sg-3" {
			t.Errorf("SecretGroupID = %q, want sg-3", data.SecretGroupID.ValueString())
		}
		if data.Name.ValueString() != "my-keystore" {
			t.Errorf("Name = %q, want my-keystore", data.Name.ValueString())
		}
		if data.Type.ValueString() != "PEM" {
			t.Errorf("Type = %q, want PEM", data.Type.ValueString())
		}
		if data.ExpirationDate.ValueString() != "" {
			t.Errorf("ExpirationDate = %q, want empty", data.ExpirationDate.ValueString())
		}
	})

	t.Run("expiration and algorithm are populated when set", func(t *testing.T) {
		ks := &secretsmgmt.KeystoreResponse{
			Name:           "jks-ks",
			Type:           "JKS",
			Meta:           secretsmgmt.SecretGroupMeta{ID: "ks-uuid-2"},
			ExpirationDate: "2030-01-01",
			Algorithm:      "RSA",
		}
		data := &KeystoreResourceModel{}
		r.flattenKeystore(ks, data, "o", "e", "s")
		if data.ExpirationDate.ValueString() != "2030-01-01" {
			t.Errorf("ExpirationDate = %q, want 2030-01-01", data.ExpirationDate.ValueString())
		}
		if data.Algorithm.ValueString() != "RSA" {
			t.Errorf("Algorithm = %q, want RSA", data.Algorithm.ValueString())
		}
	})
}

func TestKeystoreResource_expandRequest(t *testing.T) {
	r := &KeystoreResource{}

	t.Run("PEM type with base64 certificate is decoded", func(t *testing.T) {
		certPEM := "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----"
		certB64 := base64.StdEncoding.EncodeToString([]byte(certPEM))
		data := &KeystoreResourceModel{
			Name:            types.StringValue("test-ks"),
			Type:            types.StringValue("PEM"),
			CertificateB64:  types.StringValue(certB64),
			KeyB64:          types.StringNull(),
			KeystoreFileB64: types.StringNull(),
			StorePassphrase: types.StringValue(""),
			KeyPassphrase:   types.StringValue("secret"),
			Alias:           types.StringValue(""),
			CaPathB64:       types.StringNull(),
		}
		req, err := r.expandRequest(data)
		if err != nil {
			t.Fatalf("expandRequest() unexpected error: %v", err)
		}
		if req.Name != "test-ks" {
			t.Errorf("Name = %q, want test-ks", req.Name)
		}
		if req.Type != "PEM" {
			t.Errorf("Type = %q, want PEM", req.Type)
		}
		if string(req.Certificate) != certPEM {
			t.Errorf("Certificate bytes mismatch")
		}
	})

	t.Run("JKS type with base64 keystore file is decoded", func(t *testing.T) {
		ksBytes := []byte("fake-jks-content")
		ksB64 := base64.StdEncoding.EncodeToString(ksBytes)
		data := &KeystoreResourceModel{
			Name:            types.StringValue("jks-ks"),
			Type:            types.StringValue("JKS"),
			CertificateB64:  types.StringNull(),
			KeyB64:          types.StringNull(),
			KeystoreFileB64: types.StringValue(ksB64),
			StorePassphrase: types.StringValue("store-pass"),
			KeyPassphrase:   types.StringValue("key-pass"),
			Alias:           types.StringValue("my-alias"),
			CaPathB64:       types.StringNull(),
		}
		req, err := r.expandRequest(data)
		if err != nil {
			t.Fatalf("expandRequest() unexpected error: %v", err)
		}
		if string(req.Keystore) != string(ksBytes) {
			t.Error("Keystore bytes mismatch")
		}
		if req.StorePassphrase != "store-pass" {
			t.Errorf("StorePassphrase = %q, want store-pass", req.StorePassphrase)
		}
		if req.Alias != "my-alias" {
			t.Errorf("Alias = %q, want my-alias", req.Alias)
		}
	})

	t.Run("invalid base64 certificate returns error", func(t *testing.T) {
		data := &KeystoreResourceModel{
			Name:            types.StringValue("bad"),
			Type:            types.StringValue("PEM"),
			CertificateB64:  types.StringValue("not-valid-base64!!!"),
			KeyB64:          types.StringNull(),
			KeystoreFileB64: types.StringNull(),
			StorePassphrase: types.StringValue(""),
			KeyPassphrase:   types.StringValue(""),
			Alias:           types.StringValue(""),
			CaPathB64:       types.StringNull(),
		}
		_, err := r.expandRequest(data)
		if err == nil {
			t.Error("Expected error for invalid base64 certificate")
		}
	})

	t.Run("ca_path_base64 is decoded when set", func(t *testing.T) {
		caBytes := []byte("ca-cert-content")
		caB64 := base64.StdEncoding.EncodeToString(caBytes)
		data := &KeystoreResourceModel{
			Name:            types.StringValue("ks-with-ca"),
			Type:            types.StringValue("PEM"),
			CertificateB64:  types.StringNull(),
			KeyB64:          types.StringNull(),
			KeystoreFileB64: types.StringNull(),
			StorePassphrase: types.StringValue(""),
			KeyPassphrase:   types.StringValue(""),
			Alias:           types.StringValue(""),
			CaPathB64:       types.StringValue(caB64),
		}
		req, err := r.expandRequest(data)
		if err != nil {
			t.Fatalf("expandRequest() unexpected error: %v", err)
		}
		if string(req.CaPath) != string(caBytes) {
			t.Error("CaPath bytes mismatch")
		}
	})

	t.Run("PKCS12 type uses keystore field", func(t *testing.T) {
		ksBytes := []byte("pkcs12-content")
		ksB64 := base64.StdEncoding.EncodeToString(ksBytes)
		data := &KeystoreResourceModel{
			Name:            types.StringValue("pkcs12-ks"),
			Type:            types.StringValue("PKCS12"),
			CertificateB64:  types.StringNull(),
			KeyB64:          types.StringNull(),
			KeystoreFileB64: types.StringValue(ksB64),
			StorePassphrase: types.StringValue(""),
			KeyPassphrase:   types.StringValue(""),
			Alias:           types.StringValue(""),
			CaPathB64:       types.StringNull(),
		}
		req, err := r.expandRequest(data)
		if err != nil {
			t.Fatalf("expandRequest() unexpected error: %v", err)
		}
		if string(req.Keystore) != string(ksBytes) {
			t.Error("Keystore bytes mismatch for PKCS12")
		}
	})
}

// ========= TruststoreResource =========

func TestTruststoreResource_flattenTruststore(t *testing.T) {
	r := &TruststoreResource{}

	t.Run("basic truststore is flattened", func(t *testing.T) {
		ts := &secretsmgmt.TruststoreResponse{
			Name: "my-truststore",
			Type: "PEM",
			Meta: secretsmgmt.SecretGroupMeta{ID: "ts-uuid-1"},
		}
		data := &TruststoreResourceModel{}
		r.flattenTruststore(ts, data, "org-1", "env-2", "sg-3")

		if data.ID.ValueString() != "ts-uuid-1" {
			t.Errorf("ID = %q, want ts-uuid-1", data.ID.ValueString())
		}
		if data.Name.ValueString() != "my-truststore" {
			t.Errorf("Name = %q, want my-truststore", data.Name.ValueString())
		}
		if data.Type.ValueString() != "PEM" {
			t.Errorf("Type = %q, want PEM", data.Type.ValueString())
		}
		if data.ExpirationDate.ValueString() != "" {
			t.Errorf("ExpirationDate should be empty, got %q", data.ExpirationDate.ValueString())
		}
	})

	t.Run("expiration and algorithm are populated", func(t *testing.T) {
		ts := &secretsmgmt.TruststoreResponse{
			Name:           "ts-with-dates",
			Type:           "JKS",
			Meta:           secretsmgmt.SecretGroupMeta{ID: "ts-2"},
			ExpirationDate: "2025-12-31",
			Algorithm:      "SHA256",
		}
		data := &TruststoreResourceModel{}
		r.flattenTruststore(ts, data, "o", "e", "s")
		if data.ExpirationDate.ValueString() != "2025-12-31" {
			t.Errorf("ExpirationDate = %q, want 2025-12-31", data.ExpirationDate.ValueString())
		}
		if data.Algorithm.ValueString() != "SHA256" {
			t.Errorf("Algorithm = %q, want SHA256", data.Algorithm.ValueString())
		}
	})
}

func TestTruststoreResource_expandRequest(t *testing.T) {
	r := &TruststoreResource{}

	t.Run("name and type are set", func(t *testing.T) {
		data := &TruststoreResourceModel{
			Name:          types.StringValue("my-ts"),
			Type:          types.StringValue("PEM"),
			Passphrase:    types.StringValue(""),
			TrustStoreB64: types.StringNull(),
		}
		req, err := r.expandRequest(data)
		if err != nil {
			t.Fatalf("expandRequest() unexpected error: %v", err)
		}
		if req.Name != "my-ts" {
			t.Errorf("Name = %q, want my-ts", req.Name)
		}
		if req.Type != "PEM" {
			t.Errorf("Type = %q, want PEM", req.Type)
		}
		if req.TrustStore != nil {
			t.Errorf("TrustStore should be nil when no base64 provided")
		}
	})

	t.Run("truststore_base64 is decoded", func(t *testing.T) {
		tsBytes := []byte("truststore-content")
		tsB64 := base64.StdEncoding.EncodeToString(tsBytes)
		data := &TruststoreResourceModel{
			Name:          types.StringValue("ts"),
			Type:          types.StringValue("JKS"),
			Passphrase:    types.StringValue("pass"),
			TrustStoreB64: types.StringValue(tsB64),
		}
		req, err := r.expandRequest(data)
		if err != nil {
			t.Fatalf("expandRequest() unexpected error: %v", err)
		}
		if string(req.TrustStore) != string(tsBytes) {
			t.Error("TrustStore bytes mismatch")
		}
		if req.Passphrase != "pass" {
			t.Errorf("Passphrase = %q, want pass", req.Passphrase)
		}
	})

	t.Run("invalid base64 returns error", func(t *testing.T) {
		data := &TruststoreResourceModel{
			Name:          types.StringValue("bad"),
			Type:          types.StringValue("PEM"),
			Passphrase:    types.StringValue(""),
			TrustStoreB64: types.StringValue("!!!not-base64"),
		}
		_, err := r.expandRequest(data)
		if err == nil {
			t.Error("Expected error for invalid base64")
		}
	})
}

// ========= CertificateResource =========

func TestCertificateResource_flatten(t *testing.T) {
	r := &CertificateResource{}

	t.Run("basic certificate is flattened", func(t *testing.T) {
		cert := &secretsmgmt.CertificateResponse{
			Name: "my-cert",
			Type: "PEM",
			Meta: secretsmgmt.SecretGroupMeta{ID: "cert-uuid-1"},
		}
		data := &CertificateResourceModel{}
		r.flatten(cert, data, "org-1", "env-2", "sg-3")

		if data.ID.ValueString() != "cert-uuid-1" {
			t.Errorf("ID = %q, want cert-uuid-1", data.ID.ValueString())
		}
		if data.Name.ValueString() != "my-cert" {
			t.Errorf("Name = %q, want my-cert", data.Name.ValueString())
		}
		if data.ExpirationDate.ValueString() != "" {
			t.Errorf("ExpirationDate should be empty, got %q", data.ExpirationDate.ValueString())
		}
	})

	t.Run("expiration and algorithm are populated", func(t *testing.T) {
		cert := &secretsmgmt.CertificateResponse{
			Name:           "cert-with-dates",
			Type:           "PEM",
			Meta:           secretsmgmt.SecretGroupMeta{ID: "cert-2"},
			ExpirationDate: "2028-06-30",
			Algorithm:      "EC",
		}
		data := &CertificateResourceModel{}
		r.flatten(cert, data, "o", "e", "s")
		if data.ExpirationDate.ValueString() != "2028-06-30" {
			t.Errorf("ExpirationDate = %q, want 2028-06-30", data.ExpirationDate.ValueString())
		}
		if data.Algorithm.ValueString() != "EC" {
			t.Errorf("Algorithm = %q, want EC", data.Algorithm.ValueString())
		}
	})
}

// ========= CertificatePinsetResource =========

func TestCertificatePinsetResource_flatten(t *testing.T) {
	r := &CertificatePinsetResource{}

	t.Run("basic pinset is flattened", func(t *testing.T) {
		pin := &secretsmgmt.CertificatePinsetResponse{
			Name: "my-pinset",
			Meta: secretsmgmt.SecretGroupMeta{ID: "pin-uuid-1"},
		}
		data := &CertificatePinsetResourceModel{}
		r.flatten(pin, data, "org-1", "env-2", "sg-3")

		if data.ID.ValueString() != "pin-uuid-1" {
			t.Errorf("ID = %q, want pin-uuid-1", data.ID.ValueString())
		}
		if data.Name.ValueString() != "my-pinset" {
			t.Errorf("Name = %q, want my-pinset", data.Name.ValueString())
		}
		if data.SecretGroupID.ValueString() != "sg-3" {
			t.Errorf("SecretGroupID = %q, want sg-3", data.SecretGroupID.ValueString())
		}
		if data.ExpirationDate.ValueString() != "" {
			t.Errorf("ExpirationDate should be empty, got %q", data.ExpirationDate.ValueString())
		}
	})

	t.Run("expiration and algorithm are populated", func(t *testing.T) {
		pin := &secretsmgmt.CertificatePinsetResponse{
			Name:           "pin-with-algo",
			Meta:           secretsmgmt.SecretGroupMeta{ID: "pin-2"},
			ExpirationDate: "2027-01-01",
			Algorithm:      "RSA",
		}
		data := &CertificatePinsetResourceModel{}
		r.flatten(pin, data, "o", "e", "s")
		if data.ExpirationDate.ValueString() != "2027-01-01" {
			t.Errorf("ExpirationDate = %q, want 2027-01-01", data.ExpirationDate.ValueString())
		}
		if data.Algorithm.ValueString() != "RSA" {
			t.Errorf("Algorithm = %q, want RSA", data.Algorithm.ValueString())
		}
	})
}

// ========= SecretGroupResource =========

func TestSecretGroupResource_flattenSecretGroup(t *testing.T) {
	r := &SecretGroupResource{}

	t.Run("basic secret group is flattened", func(t *testing.T) {
		sg := &secretsmgmt.SecretGroupResponse{
			Name:         "my-sg",
			Downloadable: true,
			Meta:         secretsmgmt.SecretGroupMeta{ID: "sg-uuid-1"},
			CurrentState: "active",
		}
		data := &SecretGroupResourceModel{}
		r.flattenSecretGroup(sg, data, "org-1", "env-2")

		if data.ID.ValueString() != "sg-uuid-1" {
			t.Errorf("ID = %q, want sg-uuid-1", data.ID.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", data.EnvironmentID.ValueString())
		}
		if data.Name.ValueString() != "my-sg" {
			t.Errorf("Name = %q, want my-sg", data.Name.ValueString())
		}
		if !data.Downloadable.ValueBool() {
			t.Error("Downloadable = false, want true")
		}
		if data.CurrentState.ValueString() != "active" {
			t.Errorf("CurrentState = %q, want active", data.CurrentState.ValueString())
		}
	})

	t.Run("empty current_state defaults to active", func(t *testing.T) {
		sg := &secretsmgmt.SecretGroupResponse{
			Name:         "sg2",
			Meta:         secretsmgmt.SecretGroupMeta{ID: "sg-2"},
			CurrentState: "",
		}
		data := &SecretGroupResourceModel{}
		r.flattenSecretGroup(sg, data, "o", "e")
		if data.CurrentState.ValueString() != "active" {
			t.Errorf("CurrentState = %q, want active (default)", data.CurrentState.ValueString())
		}
	})
}

// ========= SharedSecretResource =========

func TestSharedSecretResource_expand(t *testing.T) {
	r := &SharedSecretResource{}

	t.Run("UsernamePassword type includes credentials", func(t *testing.T) {
		data := &SharedSecretResourceModel{
			Name:     types.StringValue("my-secret"),
			Type:     types.StringValue("UsernamePassword"),
			Username: types.StringValue("admin"),
			Password: types.StringValue("s3cr3t"),
		}
		result := r.expand(data)
		if result.Name != "my-secret" {
			t.Errorf("Name = %q, want my-secret", result.Name)
		}
		if result.Type != "UsernamePassword" {
			t.Errorf("Type = %q, want UsernamePassword", result.Type)
		}
		if result.Username != "admin" {
			t.Errorf("Username = %q, want admin", result.Username)
		}
		if result.Password != "s3cr3t" {
			t.Errorf("Password = %q, want s3cr3t", result.Password)
		}
	})

	t.Run("S3Credential type includes access key", func(t *testing.T) {
		data := &SharedSecretResourceModel{
			Name:            types.StringValue("s3-cred"),
			Type:            types.StringValue("S3Credential"),
			AccessKeyID:     types.StringValue("AKIAEXAMPLE"),
			SecretAccessKey: types.StringValue("secret-key"),
		}
		result := r.expand(data)
		if result.AccessKeyID != "AKIAEXAMPLE" {
			t.Errorf("AccessKeyID = %q, want AKIAEXAMPLE", result.AccessKeyID)
		}
		if result.SecretAccessKey != "secret-key" {
			t.Errorf("SecretAccessKey = %q, want secret-key", result.SecretAccessKey)
		}
	})

	t.Run("SymmetricKey type includes key", func(t *testing.T) {
		data := &SharedSecretResourceModel{
			Name: types.StringValue("sym-key"),
			Type: types.StringValue("SymmetricKey"),
			Key:  types.StringValue("mykey123"),
		}
		result := r.expand(data)
		if result.Key != "mykey123" {
			t.Errorf("Key = %q, want mykey123", result.Key)
		}
	})

	t.Run("Blob type includes content", func(t *testing.T) {
		data := &SharedSecretResourceModel{
			Name:    types.StringValue("my-blob"),
			Type:    types.StringValue("Blob"),
			Content: types.StringValue("blob-content"),
		}
		result := r.expand(data)
		if result.Content != "blob-content" {
			t.Errorf("Content = %q, want blob-content", result.Content)
		}
	})

	t.Run("expiration date is included when set", func(t *testing.T) {
		data := &SharedSecretResourceModel{
			Name:           types.StringValue("expiring"),
			Type:           types.StringValue("Blob"),
			ExpirationDate: types.StringValue("2030-01-01"),
			Content:        types.StringValue("data"),
		}
		result := r.expand(data)
		if result.ExpirationDate != "2030-01-01" {
			t.Errorf("ExpirationDate = %q, want 2030-01-01", result.ExpirationDate)
		}
	})

	t.Run("null expiration date is omitted", func(t *testing.T) {
		data := &SharedSecretResourceModel{
			Name:           types.StringValue("no-exp"),
			Type:           types.StringValue("Blob"),
			ExpirationDate: types.StringNull(),
			Content:        types.StringValue("data"),
		}
		result := r.expand(data)
		if result.ExpirationDate != "" {
			t.Errorf("ExpirationDate = %q, want empty", result.ExpirationDate)
		}
	})
}

func TestSharedSecretResource_flatten(t *testing.T) {
	r := &SharedSecretResource{}

	t.Run("basic shared secret is flattened", func(t *testing.T) {
		ss := &secretsmgmt.SharedSecretResponse{
			Name: "my-secret",
			Type: "UsernamePassword",
			Meta: secretsmgmt.SecretGroupMeta{ID: "ss-uuid-1"},
		}
		data := &SharedSecretResourceModel{}
		r.flatten(ss, data, "org-1", "env-2", "sg-3")

		if data.ID.ValueString() != "ss-uuid-1" {
			t.Errorf("ID = %q, want ss-uuid-1", data.ID.ValueString())
		}
		if data.Name.ValueString() != "my-secret" {
			t.Errorf("Name = %q, want my-secret", data.Name.ValueString())
		}
		if data.Type.ValueString() != "UsernamePassword" {
			t.Errorf("Type = %q, want UsernamePassword", data.Type.ValueString())
		}
		if data.SecretGroupID.ValueString() != "sg-3" {
			t.Errorf("SecretGroupID = %q, want sg-3", data.SecretGroupID.ValueString())
		}
	})

	t.Run("username from API is populated", func(t *testing.T) {
		ss := &secretsmgmt.SharedSecretResponse{
			Name:     "user-secret",
			Type:     "UsernamePassword",
			Meta:     secretsmgmt.SecretGroupMeta{ID: "ss-2"},
			Username: "returned-user",
		}
		data := &SharedSecretResourceModel{}
		r.flatten(ss, data, "o", "e", "s")
		if data.Username.ValueString() != "returned-user" {
			t.Errorf("Username = %q, want returned-user", data.Username.ValueString())
		}
	})

	t.Run("access key from API is populated", func(t *testing.T) {
		ss := &secretsmgmt.SharedSecretResponse{
			Name:        "s3-secret",
			Type:        "S3Credential",
			Meta:        secretsmgmt.SecretGroupMeta{ID: "ss-3"},
			AccessKeyID: "RETURNED-KEY",
		}
		data := &SharedSecretResourceModel{}
		r.flatten(ss, data, "o", "e", "s")
		if data.AccessKeyID.ValueString() != "RETURNED-KEY" {
			t.Errorf("AccessKeyID = %q, want RETURNED-KEY", data.AccessKeyID.ValueString())
		}
	})

	t.Run("expiration date is set when present", func(t *testing.T) {
		ss := &secretsmgmt.SharedSecretResponse{
			Name:           "expiring",
			Type:           "Blob",
			Meta:           secretsmgmt.SecretGroupMeta{ID: "ss-4"},
			ExpirationDate: "2030-06-01",
		}
		data := &SharedSecretResourceModel{}
		r.flatten(ss, data, "o", "e", "s")
		if data.ExpirationDate.ValueString() != "2030-06-01" {
			t.Errorf("ExpirationDate = %q, want 2030-06-01", data.ExpirationDate.ValueString())
		}
	})

	t.Run("empty expiration date becomes empty string", func(t *testing.T) {
		ss := &secretsmgmt.SharedSecretResponse{
			Name: "no-exp",
			Type: "Blob",
			Meta: secretsmgmt.SecretGroupMeta{ID: "ss-5"},
		}
		data := &SharedSecretResourceModel{}
		r.flatten(ss, data, "o", "e", "s")
		if data.ExpirationDate.ValueString() != "" {
			t.Errorf("ExpirationDate = %q, want empty", data.ExpirationDate.ValueString())
		}
	})
}
