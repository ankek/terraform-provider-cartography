package provider

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DiagramDataSource{}

// DiagramDataSource defines the data source implementation.
type DiagramDataSource struct {
	generator *DiagramGenerator
}

func NewDiagramDataSource() datasource.DataSource {
	return &DiagramDataSource{
		generator: &DiagramGenerator{},
	}
}

// DiagramDataSourceModel describes the data source data model.
type DiagramDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	StatePath     types.String `tfsdk:"state_path"`
	ConfigPath    types.String `tfsdk:"config_path"`
	OutputPath    types.String `tfsdk:"output_path"`
	Format        types.String `tfsdk:"format"`
	Direction     types.String `tfsdk:"direction"`
	IncludeLabels types.Bool   `tfsdk:"include_labels"`
	Title         types.String `tfsdk:"title"`
	UseIcons      types.Bool   `tfsdk:"use_icons"`
	ResourceCount types.Int64  `tfsdk:"resource_count"`
}

func (d *DiagramDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_diagram"
}

func (d *DiagramDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads Terraform state or configuration and generates infrastructure diagrams.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Data source identifier",
			},
			"state_path": schema.StringAttribute{
				MarkdownDescription: "Path to terraform.tfstate file. If not provided, will attempt to read from config_path.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("config_path")),
				},
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to directory containing .tf files. Used when state_path is not available.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("state_path")),
				},
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path where the diagram will be saved.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "Output format: 'svg', 'png', 'jpg', or 'jpeg'. Default is 'svg'. Note: PNG and JPEG export requires resvg, inkscape, or imagemagick to be installed for high quality output.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("svg", "png", "jpg", "jpeg"),
				},
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "Diagram direction: 'TB' (top to bottom), 'LR' (left to right), 'BT' (bottom to top), or 'RL' (right to left). Default is 'TB'.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("TB", "LR", "BT", "RL"),
				},
			},
			"include_labels": schema.BoolAttribute{
				MarkdownDescription: "Include resource names and attributes as labels. Default is true.",
				Optional:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Title for the diagram.",
				Optional:            true,
			},
			"use_icons": schema.BoolAttribute{
				MarkdownDescription: "Use official cloud provider icons if available. Falls back to colored boxes if icons not found. Default is false.",
				Optional:            true,
			},
			"resource_count": schema.Int64Attribute{
				MarkdownDescription: "Number of resources in the diagram.",
				Computed:            true,
			},
		},
	}
}

func (d *DiagramDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
}

func (d *DiagramDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DiagramDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults
	format := "svg"
	if !data.Format.IsNull() && data.Format.ValueString() != "" {
		format = data.Format.ValueString()
	}
	data.Format = types.StringValue(format)

	direction := "TB"
	if !data.Direction.IsNull() && data.Direction.ValueString() != "" {
		direction = data.Direction.ValueString()
	}
	data.Direction = types.StringValue(direction)

	includeLabels := true
	if !data.IncludeLabels.IsNull() {
		includeLabels = data.IncludeLabels.ValueBool()
	}
	data.IncludeLabels = types.BoolValue(includeLabels)

	useIcons := false
	if !data.UseIcons.IsNull() {
		useIcons = data.UseIcons.ValueBool()
	}

	// Use the generator to create the diagram
	result, err := d.generator.Generate(ctx, DiagramConfig{
		StatePath:     data.StatePath.ValueString(),
		ConfigPath:    data.ConfigPath.ValueString(),
		OutputPath:    data.OutputPath.ValueString(),
		Format:        data.Format.ValueString(),
		Direction:     data.Direction.ValueString(),
		IncludeLabels: data.IncludeLabels.ValueBool(),
		Title:         data.Title.ValueString(),
		UseIcons:      useIcons,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate diagram", err.Error())
		return
	}

	// Set resource count from result
	data.ResourceCount = types.Int64Value(result.ResourceCount)

	// Generate ID based on content
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s_%s_%s", data.OutputPath.ValueString(), format, direction)))
	data.ID = types.StringValue(fmt.Sprintf("%x", hash[:8]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
