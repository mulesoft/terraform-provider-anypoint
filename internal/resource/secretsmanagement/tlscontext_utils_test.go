package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestExtractIDFromPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keystore path",
			input: "keystores/abc-123-def",
			want:  "abc-123-def",
		},
		{
			name:  "truststore path",
			input: "truststores/xyz-789",
			want:  "xyz-789",
		},
		{
			name:  "plain ID no slash",
			input: "plain-id",
			want:  "plain-id",
		},
		{
			name:  "nested path returns last segment",
			input: "a/b/c/d/final-id",
			want:  "final-id",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIDFromPath(tt.input)
			if result != tt.want {
				t.Errorf("extractIDFromPath(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestTLSContextResource_expandTLSContext(t *testing.T) {
	r := &TLSContextResource{}
	ctx := context.Background()

	t.Run("OmniGateway target is translated to FlexGateway", func(t *testing.T) {
		data := &TLSContextResourceModel{
			Name:                       types.StringValue("test-tls"),
			Target:                     types.StringValue("OmniGateway"),
			KeystoreID:                 types.StringNull(),
			TruststoreID:               types.StringNull(),
			MinTLSVersion:              types.StringNull(),
			MaxTLSVersion:              types.StringNull(),
			AlpnProtocols:              types.ListNull(types.StringType),
			CipherSuites:               types.ListNull(types.StringType),
			EnableClientCertValidation: types.BoolValue(false),
			SkipServerCertValidation:   types.BoolValue(false),
		}
		result := r.expandTLSContext(ctx, data, nil)
		if result.Target != "FlexGateway" {
			t.Errorf("Target = %s, want FlexGateway", result.Target)
		}
	})

	t.Run("non-OmniGateway target is passed through", func(t *testing.T) {
		data := &TLSContextResourceModel{
			Name:                       types.StringValue("test-tls"),
			Target:                     types.StringValue("Mule"),
			KeystoreID:                 types.StringNull(),
			TruststoreID:               types.StringNull(),
			MinTLSVersion:              types.StringNull(),
			MaxTLSVersion:              types.StringNull(),
			AlpnProtocols:              types.ListNull(types.StringType),
			CipherSuites:               types.ListNull(types.StringType),
			EnableClientCertValidation: types.BoolValue(false),
			SkipServerCertValidation:   types.BoolValue(false),
		}
		result := r.expandTLSContext(ctx, data, nil)
		if result.Target != "Mule" {
			t.Errorf("Target = %s, want Mule", result.Target)
		}
	})

	t.Run("keystore ID is converted to path reference", func(t *testing.T) {
		data := &TLSContextResourceModel{
			Name:                       types.StringValue("test-tls"),
			Target:                     types.StringValue("Mule"),
			KeystoreID:                 types.StringValue("ks-uuid-123"),
			TruststoreID:               types.StringNull(),
			MinTLSVersion:              types.StringNull(),
			MaxTLSVersion:              types.StringNull(),
			AlpnProtocols:              types.ListNull(types.StringType),
			CipherSuites:               types.ListNull(types.StringType),
			EnableClientCertValidation: types.BoolValue(false),
			SkipServerCertValidation:   types.BoolValue(false),
		}
		result := r.expandTLSContext(ctx, data, nil)
		if result.Keystore == nil {
			t.Fatal("Keystore should be set")
		}
		if result.Keystore.Path != "keystores/ks-uuid-123" {
			t.Errorf("Keystore.Path = %s, want keystores/ks-uuid-123", result.Keystore.Path)
		}
	})

	t.Run("truststore ID is converted to path reference", func(t *testing.T) {
		data := &TLSContextResourceModel{
			Name:                       types.StringValue("test-tls"),
			Target:                     types.StringValue("Mule"),
			KeystoreID:                 types.StringNull(),
			TruststoreID:               types.StringValue("ts-uuid-456"),
			MinTLSVersion:              types.StringNull(),
			MaxTLSVersion:              types.StringNull(),
			AlpnProtocols:              types.ListNull(types.StringType),
			CipherSuites:               types.ListNull(types.StringType),
			EnableClientCertValidation: types.BoolValue(true),
			SkipServerCertValidation:   types.BoolValue(true),
		}
		result := r.expandTLSContext(ctx, data, nil)
		if result.Truststore == nil {
			t.Fatal("Truststore should be set")
		}
		if result.Truststore.Path != "truststores/ts-uuid-456" {
			t.Errorf("Truststore.Path = %s, want truststores/ts-uuid-456", result.Truststore.Path)
		}
		if !result.InboundSettings.EnableClientCertValidation {
			t.Error("EnableClientCertValidation should be true")
		}
		if !result.OutboundSettings.SkipServerCertValidation {
			t.Error("SkipServerCertValidation should be true")
		}
	})

	t.Run("TLS version and protocols are included when set", func(t *testing.T) {
		alpnList, _ := types.ListValueFrom(ctx, types.StringType, []string{"h2", "http/1.1"})
		cipherList, _ := types.ListValueFrom(ctx, types.StringType, []string{"TLS_AES_128_GCM_SHA256"})

		data := &TLSContextResourceModel{
			Name:                       types.StringValue("test-tls"),
			Target:                     types.StringValue("Mule"),
			KeystoreID:                 types.StringNull(),
			TruststoreID:               types.StringNull(),
			MinTLSVersion:              types.StringValue("TLSv1.2"),
			MaxTLSVersion:              types.StringValue("TLSv1.3"),
			AlpnProtocols:              alpnList,
			CipherSuites:               cipherList,
			EnableClientCertValidation: types.BoolValue(false),
			SkipServerCertValidation:   types.BoolValue(false),
		}
		result := r.expandTLSContext(ctx, data, nil)
		if result.MinTLSVersion != "TLSv1.2" {
			t.Errorf("MinTLSVersion = %s, want TLSv1.2", result.MinTLSVersion)
		}
		if result.MaxTLSVersion != "TLSv1.3" {
			t.Errorf("MaxTLSVersion = %s, want TLSv1.3", result.MaxTLSVersion)
		}
		if len(result.AlpnProtocols) != 2 {
			t.Errorf("AlpnProtocols len = %d, want 2", len(result.AlpnProtocols))
		}
		if len(result.CipherSuites) != 1 {
			t.Errorf("CipherSuites len = %d, want 1", len(result.CipherSuites))
		}
	})
}
