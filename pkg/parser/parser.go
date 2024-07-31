package parser

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

func ParseTerraformConfig(filePath string) (*hcl.File, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filePath)

	if diags.HasErrors() {
		return nil, diags
	}
	return file, nil
}
