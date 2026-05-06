package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	dsAccessManagement "github.com/mulesoft/terraform-provider-anypoint/internal/datasource/accessmanagement"
	dsAgentsTools "github.com/mulesoft/terraform-provider-anypoint/internal/datasource/agentstools"
	dsApiManagement "github.com/mulesoft/terraform-provider-anypoint/internal/datasource/apimanagement"
	dsCloudHub2 "github.com/mulesoft/terraform-provider-anypoint/internal/datasource/cloudhub2"
	dsSecretsManagement "github.com/mulesoft/terraform-provider-anypoint/internal/datasource/secretsmanagement"
	resourceAccessManagement "github.com/mulesoft/terraform-provider-anypoint/internal/resource/accessmanagement"
	resourceAgentsTools "github.com/mulesoft/terraform-provider-anypoint/internal/resource/agentstools"
	resourceApiManagement "github.com/mulesoft/terraform-provider-anypoint/internal/resource/apimanagement"
	resourceCloudHub2 "github.com/mulesoft/terraform-provider-anypoint/internal/resource/cloudhub2"
	resourceSecretsManagement "github.com/mulesoft/terraform-provider-anypoint/internal/resource/secretsmanagement"
)

var (
	_ provider.Provider                   = &AnypointProvider{}
	_ provider.ProviderWithValidateConfig = &AnypointProvider{}
)

type AnypointProvider struct {
	version string
}

type AnypointProviderModel struct {
	AuthType     types.String `tfsdk:"auth_type"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	BaseURL      types.String `tfsdk:"base_url"`
	Timeout      types.Int64  `tfsdk:"timeout"`
}

func (p *AnypointProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anypoint"
	resp.Version = p.version
}

func (p *AnypointProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Anypoint Platform.",
		Attributes: map[string]schema.Attribute{
			"auth_type": schema.StringAttribute{
				MarkdownDescription: "Authentication type to use. Valid values are `connected_app` (default) for client credentials flow, or `user` for password grant flow.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("connected_app", "user"),
				},
			},
			"client_id": schema.StringAttribute{
				Description: "Anypoint Platform client ID for connected app or user authentication. May also be provided via ANYPOINT_CLIENT_ID environment variable.",
				Optional:    true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The client secret for the Anypoint Platform connected app. May also be provided via ANYPOINT_CLIENT_SECRET environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username for user authentication (only required when auth_type is 'user'). May also be provided via ANYPOINT_USERNAME environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password for user authentication (only required when auth_type is 'user'). May also be provided via ANYPOINT_PASSWORD environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "The base URL of the Anypoint Platform. Defaults to `https://anypoint.mulesoft.com`. May also be provided via ANYPOINT_BASE_URL environment variable.",
				Optional:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "The timeout for API requests in seconds. Defaults to 600.",
				Optional:            true,
			},
		},
	}
}

// stringValueOrEnv returns the config value if set, otherwise falls back to
// the environment variable. Returns "" if neither is set.
func stringValueOrEnv(val types.String, envKey string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envKey)
}

func (p *AnypointProvider) ValidateConfig(ctx context.Context, req provider.ValidateConfigRequest, resp *provider.ValidateConfigResponse) {
	var config AnypointProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When values come from variables or other expressions they are unknown at
	// validation time.  We can only validate what is statically known; anything
	// unknown will be checked later during Configure when all values are resolved.
	if config.AuthType.IsUnknown() || config.ClientID.IsUnknown() || config.ClientSecret.IsUnknown() ||
		config.Username.IsUnknown() || config.Password.IsUnknown() {
		return
	}

	authType := stringValueOrEnv(config.AuthType, "ANYPOINT_AUTH_TYPE")
	if authType == "" {
		authType = "connected_app"
	}

	clientID := stringValueOrEnv(config.ClientID, "ANYPOINT_CLIENT_ID")
	clientSecret := stringValueOrEnv(config.ClientSecret, "ANYPOINT_CLIENT_SECRET")

	switch authType {
	case "connected_app":
		if clientID == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("client_id"),
				"Missing Client ID",
				"client_id must be set in the provider configuration or via the ANYPOINT_CLIENT_ID environment variable.",
			)
		}
		if clientSecret == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("client_secret"),
				"Missing Client Secret",
				"client_secret must be set in the provider configuration or via the ANYPOINT_CLIENT_SECRET environment variable.",
			)
		}
	case "user":
		username := stringValueOrEnv(config.Username, "ANYPOINT_USERNAME")
		password := stringValueOrEnv(config.Password, "ANYPOINT_PASSWORD")
		if clientID == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("client_id"),
				"Missing Client ID",
				"client_id is required for user authentication. Set it in the provider configuration or via the ANYPOINT_CLIENT_ID environment variable.",
			)
		}
		if clientSecret == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("client_secret"),
				"Missing Client Secret",
				"client_secret is required for user authentication. Set it in the provider configuration or via the ANYPOINT_CLIENT_SECRET environment variable.",
			)
		}
		if username == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing Username",
				"username must be set when auth_type is 'user'. Set it in the provider configuration or via the ANYPOINT_USERNAME environment variable.",
			)
		}
		if password == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing Password",
				"password must be set when auth_type is 'user'. Set it in the provider configuration or via the ANYPOINT_PASSWORD environment variable.",
			)
		}
	default:
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_type"),
			"Invalid Authentication Type",
			"auth_type must be either 'connected_app' or 'user'.",
		)
	}
}

func (p *AnypointProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config AnypointProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientConfig := &client.Config{
		ClientID:     stringValueOrEnv(config.ClientID, "ANYPOINT_CLIENT_ID"),
		ClientSecret: stringValueOrEnv(config.ClientSecret, "ANYPOINT_CLIENT_SECRET"),
		Username:     stringValueOrEnv(config.Username, "ANYPOINT_USERNAME"),
		Password:     stringValueOrEnv(config.Password, "ANYPOINT_PASSWORD"),
		BaseURL:      stringValueOrEnv(config.BaseURL, "ANYPOINT_BASE_URL"),
		Timeout:      int(config.Timeout.ValueInt64()),
	}

	resp.DataSourceData = clientConfig
	resp.ResourceData = clientConfig
}

func (p *AnypointProvider) Resources(_ context.Context) []func() resource.Resource {
	resources := make([]func() resource.Resource, 0, 34+len(resourceApiManagement.KnownPolicyTypes()))
	resources = append(resources, []func() resource.Resource{
		// CloudHub 2.0 resources
		resourceCloudHub2.NewPrivateSpaceConfigResource,
		resourceCloudHub2.NewTLSContextResource,
		resourceCloudHub2.NewVPNConnectionResource,
		resourceCloudHub2.NewPrivateSpaceAssociationResource,
		resourceCloudHub2.NewPrivateSpaceUpgradeResource,
		resourceCloudHub2.NewPrivateSpaceAdvancedConfigResource,
		// Access Management resources
		resourceAccessManagement.NewEnvironmentResource,
		resourceAccessManagement.NewOrganizationResource,
		resourceAccessManagement.NewTeamResource,
		resourceAccessManagement.NewConnectedAppScopesResource,
		// API Management resources
		resourceApiManagement.NewManagedFlexGatewayResource,
		resourceApiManagement.NewAPIInstanceResource,
		resourceApiManagement.NewAPIPolicyResource,
		resourceApiManagement.NewSLATierResource,
		// Agents Tools resources
		resourceAgentsTools.NewAgentInstanceResource,
		resourceAgentsTools.NewMCPServerResource,
		// Secrets Management resources
		resourceSecretsManagement.NewSecretGroupResource,
		resourceSecretsManagement.NewKeystoreResource,
		resourceSecretsManagement.NewTruststoreResource,
		resourceSecretsManagement.NewTLSContextResource,
		resourceSecretsManagement.NewCertificateResource,
		resourceSecretsManagement.NewCertificatePinsetResource,
		resourceSecretsManagement.NewSharedSecretResource,
	}...)

	for _, policyType := range resourceApiManagement.KnownPolicyTypes() {
		resources = append(resources, resourceApiManagement.NewKnownPolicyResourceFunc(policyType))
	}

	return resources
}

func (p *AnypointProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Access Management data sources
		dsAccessManagement.NewEnvironmentDataSource,
		dsAccessManagement.NewOrganizationDataSource,
		dsAccessManagement.NewTeamDataSource,
		// CloudHub 2.0 data sources
		dsCloudHub2.NewTLSContextDataSource,
		dsCloudHub2.NewPrivateSpaceAssociationDataSource,
		dsCloudHub2.NewPrivateSpaceUpgradeDataSource,
		// API Management data sources
		dsApiManagement.NewManagedFlexGatewayDataSource,
		dsApiManagement.NewManagedFlexGatewaySingleDataSource,
		dsApiManagement.NewAPIInstanceDataSource,
		dsApiManagement.NewAPIUpstreamsDataSource,
		// Agents Tools data sources
		dsAgentsTools.NewAgentInstanceDataSource,
		dsAgentsTools.NewMCPServerDataSource,
		// Secrets Management data sources
		dsSecretsManagement.NewSecretGroupDataSource,
		dsSecretsManagement.NewKeystoreDataSource,
		dsSecretsManagement.NewTruststoreDataSource,
		dsSecretsManagement.NewCertificateDataSource,
		dsSecretsManagement.NewCertificatePinsetDataSource,
		dsSecretsManagement.NewSharedSecretDataSource,
		dsSecretsManagement.NewTLSContextDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AnypointProvider{
			version: version,
		}
	}
}
