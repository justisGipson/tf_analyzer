package analyzer

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/justisGipson/tf_analyzer/pkg/parser"
)

type Analyzer struct {
	filePath    string
	file        *hcl.File
	body        *hclsyntax.Body
	customRules []CustomRule
}

func NewAnalyzer(filePath string, customRulesPath string) (*Analyzer, error) {
	customRules, err := LoadCustomRules(customRulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom rules: %v", err)
	}

	return &Analyzer{
		filePath:    filePath,
		customRules: customRules,
	}, nil
}

func (a *Analyzer) Analyze() error {
	file, err := parser.ParseTerraformConfig(a.filePath)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Do something with the file
	a.file = file
	a.body = file.Body.(*hclsyntax.Body)

	a.applyCustomRules()

	a.checkUnusedVariables()
	a.checkMissingRequiredAttributes()
	a.checkDuplicateResourceNames()

	fmt.Println("Analyzing Terraform config...", a.filePath)

	return nil
}

func (a *Analyzer) checkUnusedVariables() {
	// Check for unused variables
	varDefs := make(map[string]struct{})
	for _, block := range a.body.Blocks {
		if block.Type == "variable" {
			varName := block.Labels[0]
			varDefs[varName] = struct{}{}
		}
	}

	// check if vars are used in configuration
	for _, attr := range a.body.Attributes {
		vars := attr.Expr.Variables()
		for _, v := range vars {
			delete(varDefs, v.RootName())
		}
	}

	// report unused variables
	for varName := range varDefs {
		fmt.Printf("Unused variable: %s\n", varName)
	}
}

func (a *Analyzer) checkMissingRequiredAttributes() {
	for _, block := range a.body.Blocks {
		if block.Type == "resource" {
			resourceType := block.Labels[0]
			resourceName := block.Labels[1]

			if requiredAttrs, ok := requiredAttributes[resourceType]; ok {
				for _, attr := range requiredAttrs {
					found := false
					for _, a := range block.Body.Attributes {
						if a.Name == attr {
							found = true
							break
						}
					}
					if !found {
						fmt.Printf("Missing required attribute '%s' in resource '%s.%s'\n", attr, resourceType, resourceName)
					}
				}
			}
		}
	}
}

func (a *Analyzer) checkDuplicateResourceNames() {
	resourceNames := make(map[string]map[string]struct{})

	for _, block := range a.body.Blocks {
		if block.Type == "resource" {
			resourceType := block.Labels[0]
			resourceName := block.Labels[1]

			if _, ok := resourceNames[resourceType]; !ok {
				resourceNames[resourceType] = make(map[string]struct{})
			}

			if _, ok := resourceNames[resourceType][resourceName]; ok {
				fmt.Printf("Duplicate resource name '%s.%s'\n", resourceType, resourceName)
			} else {
				resourceNames[resourceType][resourceName] = struct{}{}
			}
		}
	}
}

func (a *Analyzer) applyCustomRules() {
	for _, block := range a.body.Blocks {
		if block.Type == "resource" {
			resourceType := block.Labels[0]

			for _, rule := range a.customRules {
				if rule.ResourceType == resourceType {
					for _, attr := range rule.Attributes {
						found := false
						for _, a := range block.Body.Attributes {
							if a.Name == attr {
								found = true
								break
							}
						}
						if !found {
							fmt.Printf("Custom rule violation: Missing attribute '%s' in resource '%s'\n", attr, resourceType)
						}
					}
				}
			}
		}
	}
}
