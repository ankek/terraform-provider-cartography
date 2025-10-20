package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure CartographyProvider satisfies various provider interfaces.
var _ provider.Provider = &CartographyProvider{}

// CartographyProvider defines the provider implementation.
type CartographyProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// CartographyProviderModel describes the provider data model.
type CartographyProviderModel struct {
	// Authentication credentials for remote backends
	TerraformToken types.String `tfsdk:"terraform_token"`
	AWSAccessKey   types.String `tfsdk:"aws_access_key"`
	AWSSecretKey   types.String `tfsdk:"aws_secret_key"`
	AzureAccount   types.String `tfsdk:"azure_account"`
	AzureKey       types.String `tfsdk:"azure_key"`
	GCPCredentials types.String `tfsdk:"gcp_credentials"`
}

func (p *CartographyProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cartography"
	resp.Version = p.version
}

func (p *CartographyProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Cartography provider generates visual diagrams of your Terraform infrastructure, showing resources and their connections.",
		Attributes: map[string]schema.Attribute{
			"terraform_token": schema.StringAttribute{
				Description: "Terraform Cloud/Enterprise API token. Can also be set via TFE_TOKEN environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"aws_access_key": schema.StringAttribute{
				Description: "AWS access key for S3 backend. Can also be set via AWS_ACCESS_KEY_ID environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"aws_secret_key": schema.StringAttribute{
				Description: "AWS secret key for S3 backend. Can also be set via AWS_SECRET_ACCESS_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"azure_account": schema.StringAttribute{
				Description: "Azure Storage account name for azurerm backend.",
				Optional:    true,
			},
			"azure_key": schema.StringAttribute{
				Description: "Azure Storage account key for azurerm backend. Can also be set via ARM_ACCESS_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"gcp_credentials": schema.StringAttribute{
				Description: "GCP service account credentials (JSON) for GCS backend. Can also be set via GOOGLE_APPLICATION_CREDENTIALS environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *CartographyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CartographyProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Make credentials available to resources and data sources
	resp.DataSourceData = &data
	resp.ResourceData = &data
}

func (p *CartographyProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDiagramResource,
	}
}

func (p *CartographyProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDiagramDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CartographyProvider{
			version: version,
		}
	}
}
