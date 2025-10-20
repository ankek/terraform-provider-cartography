package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
	"github.com/ankek/terraform-provider-cartography/internal/parser"
	"github.com/ankek/terraform-provider-cartography/internal/renderer"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &DiagramResource{}
var _ resource.ResourceWithImportState = &DiagramResource{}

func NewDiagramResource() resource.Resource {
	return &DiagramResource{}
}

// DiagramResource defines the resource implementation.
type DiagramResource struct {
}

// DiagramResourceModel describes the resource data model.
type DiagramResourceModel struct {
	ID            types.String `tfsdk:"id"`
	StatePath     types.String `tfsdk:"state_path"`
	ConfigPath    types.String `tfsdk:"config_path"`
	OutputPath    types.String `tfsdk:"output_path"`
	Format        types.String `tfsdk:"format"`
	Direction     types.String `tfsdk:"direction"`
	IncludeLabels types.Bool   `tfsdk:"include_labels"`
	Title         types.String `tfsdk:"title"`
	UseIcons      types.Bool   `tfsdk:"use_icons"`
}

func (r *DiagramResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_diagram"
}

func (r *DiagramResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generates infrastructure diagrams from Terraform state or configuration files.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"state_path": schema.StringAttribute{
				MarkdownDescription: "Path to terraform.tfstate file. If not provided, will attempt to read from config_path.",
				Optional:            true,
			},
			"config_path": schema.StringAttribute{
				MarkdownDescription: "Path to directory containing .tf files. Used when state_path is not available.",
				Optional:            true,
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path where the diagram will be saved.",
				Required:            true,
			},
			"format": schema.StringAttribute{
				MarkdownDescription: "Output format: 'png' or 'svg'. Default is 'png'.",
				Optional:            true,
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "Diagram direction: 'TB' (top to bottom), 'LR' (left to right), 'BT' (bottom to top), or 'RL' (right to left). Default is 'TB'.",
				Optional:            true,
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
		},
	}
}

func (r *DiagramResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
}

func (r *DiagramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DiagramResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults
	if data.Format.IsNull() {
		data.Format = types.StringValue("png")
	}
	if data.Direction.IsNull() {
		data.Direction = types.StringValue("TB")
	}
	if data.IncludeLabels.IsNull() {
		data.IncludeLabels = types.BoolValue(true)
	}
	if data.UseIcons.IsNull() {
		data.UseIcons = types.BoolValue(false)
	}

	// Parse Terraform state or config
	var resources []parser.Resource
	var err error

	if !data.StatePath.IsNull() && data.StatePath.ValueString() != "" {
		resources, err = parser.ParseStateFile(data.StatePath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse state file", err.Error())
			return
		}
	} else if !data.ConfigPath.IsNull() && data.ConfigPath.ValueString() != "" {
		resources, err = parser.ParseConfigDirectory(data.ConfigPath.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse config directory", err.Error())
			return
		}
	} else {
		resp.Diagnostics.AddError("Missing input", "Either state_path or config_path must be provided")
		return
	}

	// Build resource graph
	resourceGraph := graph.BuildGraph(resources)

	// Render diagram
	renderOpts := renderer.RenderOptions{
		Format:        data.Format.ValueString(),
		Direction:     data.Direction.ValueString(),
		IncludeLabels: data.IncludeLabels.ValueBool(),
		Title:         data.Title.ValueString(),
		UseIcons:      data.UseIcons.ValueBool(),
	}

	err = renderer.RenderDiagram(resourceGraph, data.OutputPath.ValueString(), renderOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to render diagram", err.Error())
		return
	}

	// Generate ID
	data.ID = types.StringValue(fmt.Sprintf("%s_%s", data.OutputPath.ValueString(), data.Format.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiagramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DiagramResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if output file still exists
	if _, err := os.Stat(data.OutputPath.ValueString()); os.IsNotExist(err) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DiagramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DiagramResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-create the diagram with updated configuration
	r.Create(ctx, resource.CreateRequest{Plan: req.Plan}, (*resource.CreateResponse)(resp))
}

func (r *DiagramResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DiagramResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Optionally remove the generated diagram file
	// os.Remove(data.OutputPath.ValueString())
}

func (r *DiagramResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
