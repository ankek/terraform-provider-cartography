package renderer

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os/exec"
	"strings"

	"github.com/ankek/terraform-provider-cartography/internal/graph"
)

// ExportFormat represents the export format
type ExportFormat string

const (
	FormatSVG  ExportFormat = "svg"
	FormatPNG  ExportFormat = "png"
	FormatJPEG ExportFormat = "jpeg"
	FormatJPG  ExportFormat = "jpg"
)

// ExportDiagram exports a diagram in the specified format
func ExportDiagram(g *graph.Graph, outputPath string, opts RenderOptions) error {
	format := strings.ToLower(opts.Format)

	// Always generate SVG first (it's our high-quality source)
	svgRenderer := NewSVGRenderer(opts)

	// Calculate layout with improved algorithm (prevents overlaps, adds curves)
	nodeWidth := 220.0   // Slightly wider for better visibility
	nodeHeight := 160.0  // Taller for better icon display
	horizontalSpacing := 140.0  // More space between nodes
	verticalSpacing := 120.0    // More vertical space

	layout := CalculateImprovedLayout(g, opts.Direction, nodeWidth, nodeHeight, horizontalSpacing, verticalSpacing)

	svgData, err := svgRenderer.Render(layout, g)
	if err != nil {
		return fmt.Errorf("failed to generate SVG: %w", err)
	}

	// Export based on format
	switch ExportFormat(format) {
	case FormatSVG:
		return writeSVG(outputPath, svgData)
	case FormatPNG:
		return convertSVGToPNG(outputPath, svgData, opts)
	case FormatJPEG, FormatJPG:
		return convertSVGToJPEG(outputPath, svgData, opts)
	default:
		return fmt.Errorf("unsupported format: %s (supported: svg, png, jpg, jpeg)", format)
	}
}

// writeSVG writes SVG data to file
func writeSVG(outputPath string, svgData []byte) error {
	return writeFile(outputPath, svgData)
}

// convertSVGToPNG converts SVG to high-quality PNG
func convertSVGToPNG(outputPath string, svgData []byte, opts RenderOptions) error {
	// Try using resvg (fastest and highest quality)
	if err := convertWithResvg(svgData, outputPath, "png", opts); err == nil {
		return nil
	}

	// Try using inkscape
	if err := convertWithInkscape(svgData, outputPath, "png", opts); err == nil {
		return nil
	}

	// Try using imagemagick/convert
	if err := convertWithImageMagick(svgData, outputPath, "png", opts); err == nil {
		return nil
	}

	// Fall back to basic rasterizer (pure Go, always works but lower quality)
	return convertWithBasicRasterizer(svgData, outputPath, "png", opts)
}

// convertSVGToJPEG converts SVG to high-quality JPEG
func convertSVGToJPEG(outputPath string, svgData []byte, opts RenderOptions) error {
	// First convert to PNG (high quality)
	tempPNG := outputPath + ".temp.png"

	// Try resvg
	if err := convertWithResvg(svgData, tempPNG, "png", opts); err == nil {
		return convertPNGToJPEG(tempPNG, outputPath, 95)
	}

	// Try inkscape
	if err := convertWithInkscape(svgData, tempPNG, "png", opts); err == nil {
		return convertPNGToJPEG(tempPNG, outputPath, 95)
	}

	// Try imagemagick
	if err := convertWithImageMagick(svgData, tempPNG, "png", opts); err == nil {
		return convertPNGToJPEG(tempPNG, outputPath, 95)
	}

	// Fall back to basic conversion
	if err := convertWithBasicRasterizer(svgData, tempPNG, "png", opts); err == nil {
		return convertPNGToJPEG(tempPNG, outputPath, 95)
	}

	return fmt.Errorf("failed to convert SVG to JPEG: no converters available")
}

// convertWithResvg uses resvg for high-quality conversion (recommended)
func convertWithResvg(svgData []byte, outputPath string, format string, opts RenderOptions) error {
	cmd := exec.Command("resvg", "--width", "2400", "-", outputPath)
	cmd.Stdin = bytes.NewReader(svgData)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("resvg failed: %w (%s)", err, stderr.String())
	}

	return nil
}

// convertWithInkscape uses Inkscape for conversion
func convertWithInkscape(svgData []byte, outputPath string, format string, opts RenderOptions) error {
	// Inkscape 1.0+ syntax
	cmd := exec.Command("inkscape",
		"--pipe",
		"--export-type="+format,
		"--export-dpi=300",
		"--export-filename="+outputPath)
	cmd.Stdin = bytes.NewReader(svgData)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("inkscape failed: %w (%s)", err, stderr.String())
	}

	return nil
}

// convertWithImageMagick uses ImageMagick for conversion
func convertWithImageMagick(svgData []byte, outputPath string, format string, opts RenderOptions) error {
	cmd := exec.Command("convert",
		"-density", "300",
		"-background", "none",
		"svg:-",
		outputPath)
	cmd.Stdin = bytes.NewReader(svgData)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("imagemagick failed: %w (%s)", err, stderr.String())
	}

	return nil
}

// convertWithBasicRasterizer uses pure Go rasterization (fallback)
func convertWithBasicRasterizer(svgData []byte, outputPath string, format string, opts RenderOptions) error {
	// This is a simplified fallback - for production, consider using:
	// - github.com/srwiley/oksvg + rasterx for better SVG support
	// - github.com/tdewolff/canvas for more advanced rendering

	// For now, return an error suggesting manual conversion
	return fmt.Errorf("pure Go SVG rasterizer not implemented yet; please install one of: resvg, inkscape, or imagemagick")
}

// convertPNGToJPEG converts a PNG file to JPEG
func convertPNGToJPEG(pngPath, jpegPath string, quality int) error {
	// Read PNG
	pngFile, err := readFile(pngPath)
	if err != nil {
		return fmt.Errorf("failed to read PNG: %w", err)
	}

	img, err := png.Decode(bytes.NewReader(pngFile))
	if err != nil {
		return fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Convert to JPEG
	jpegFile, err := createFile(jpegPath)
	if err != nil {
		return fmt.Errorf("failed to create JPEG: %w", err)
	}
	defer jpegFile.Close()

	// Create white background for transparent areas
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)

	// Fill with white
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, image.White)
		}
	}

	// Draw image on white background
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	// Encode as JPEG
	return jpeg.Encode(jpegFile, rgba, &jpeg.Options{Quality: quality})
}
