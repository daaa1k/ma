// Package convert orchestrates MCP configuration conversion between supported
// tool formats. It decodes a source format into the canonical model, collects
// warnings from both the decode and encode phases, and produces output bytes in
// the target format.
package convert

import (
	"fmt"
	"io"

	"github.com/daaa1k/ma/internal/formats/claude"
	"github.com/daaa1k/ma/internal/formats/codex"
	"github.com/daaa1k/ma/internal/formats/copilot"
	"github.com/daaa1k/ma/internal/formats/opencode"
	"github.com/daaa1k/ma/internal/model"
)

// Format identifies a supported tool configuration format.
type Format string

const (
	FormatClaude   Format = "claude"
	FormatCodex    Format = "codex"
	FormatOpenCode Format = "opencode"
	FormatCopilot  Format = "copilot"
)

// Formats returns all supported format names for use in CLI help text.
func Formats() []Format {
	return []Format{FormatClaude, FormatCodex, FormatOpenCode, FormatCopilot}
}

// Warning is a non-fatal issue collected during a conversion.
type Warning struct {
	Phase   string // "decode" or "encode"
	Message string
}

func (w Warning) String() string {
	return fmt.Sprintf("[%s] %s", w.Phase, w.Message)
}

// Result holds the output bytes and any warnings produced by Convert.
type Result struct {
	Data     []byte
	Warnings []Warning
}

// Convert decodes input bytes from src format, then encodes into dst format.
// Non-fatal issues (e.g. dropped fields, skipped servers) are returned as
// warnings; the caller should print them to stderr.
func Convert(src, dst Format, input []byte) (*Result, error) {
	cfg, decodeWarnings, err := decode(src, input)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", src, err)
	}

	data, encodeWarnings, err := encode(dst, cfg)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", dst, err)
	}

	result := &Result{Data: data}
	for _, w := range decodeWarnings {
		result.Warnings = append(result.Warnings, Warning{Phase: "decode", Message: w})
	}
	for _, w := range encodeWarnings {
		result.Warnings = append(result.Warnings, Warning{Phase: "encode", Message: w})
	}
	return result, nil
}

func decode(f Format, data []byte) (*model.Config, []string, error) {
	switch f {
	case FormatClaude:
		cfg, err := claude.Decode(data)
		return cfg, nil, err

	case FormatCodex:
		cfg, warnings, err := codex.Decode(data)
		return cfg, warningsToStrings(warnings), err

	case FormatOpenCode:
		cfg, warnings, err := opencode.Decode(data)
		return cfg, warningsToStrings(warnings), err

	case FormatCopilot:
		cfg, warnings, err := copilot.Decode(data)
		return cfg, warningsToStrings(warnings), err

	default:
		return nil, nil, fmt.Errorf("unknown source format %q", f)
	}
}

func encode(f Format, cfg *model.Config) ([]byte, []string, error) {
	switch f {
	case FormatClaude:
		data, err := claude.Encode(cfg)
		return data, nil, err

	case FormatCodex:
		data, warnings, err := codex.Encode(cfg)
		return data, warningsToStrings(warnings), err

	case FormatOpenCode:
		data, warnings, err := opencode.Encode(cfg)
		return data, warningsToStrings(warnings), err

	case FormatCopilot:
		data, warnings, err := copilot.Encode(cfg)
		return data, warningsToStrings(warnings), err

	default:
		return nil, nil, fmt.Errorf("unknown target format %q", f)
	}
}

// Encode serializes a canonical config into the given format and returns the
// result bytes plus any non-fatal warnings. It is the counterpart of Decode and
// is used by the launcher package to avoid an unnecessary JSON round-trip.
func Encode(dst Format, cfg *model.Config) (*Result, error) {
	data, encodeWarnings, err := encode(dst, cfg)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", dst, err)
	}
	result := &Result{Data: data}
	for _, w := range encodeWarnings {
		result.Warnings = append(result.Warnings, Warning{Phase: "encode", Message: w})
	}
	return result, nil
}

// Decode parses input bytes in src format and returns the canonical config plus
// any non-fatal warnings. It is the counterpart of Encode.
func Decode(src Format, input []byte) (*model.Config, *Result, error) {
	cfg, decodeWarnings, err := decode(src, input)
	if err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", src, err)
	}
	result := &Result{}
	for _, w := range decodeWarnings {
		result.Warnings = append(result.Warnings, Warning{Phase: "decode", Message: w})
	}
	return cfg, result, nil
}

// WriteWarnings writes all warnings to w, prefixed with "warning: ".
func WriteWarnings(w io.Writer, warnings []Warning) {
	for _, warn := range warnings {
		_, _ = fmt.Fprintf(w, "warning: %s\n", warn)
	}
}

// warningStringer is the common interface for format-specific warning types.
type warningStringer interface {
	Error() string
}

func warningsToStrings[W warningStringer](ws []W) []string {
	if len(ws) == 0 {
		return nil
	}
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.Error()
	}
	return out
}
