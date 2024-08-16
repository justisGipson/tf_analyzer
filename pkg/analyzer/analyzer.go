package analyzer

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/justisGipson/tf_analyzer/pkg/parser"
)

type Analyzer struct {
	filePath       string
	file           *hcl.File
	body           *hclsyntax.Body
	customRules    []CustomRule
	modules        map[string]*Analyzer
	issues         []Issue
	variables      map[string]bool
	usedVariables  map[string]bool
	definedModules map[string]bool
}

type Issue struct {
	Module  string
	Message string
}

func NewAnalyzer(filePath string, customRulesPath string) (*Analyzer, error) {
	customRules, err := LoadCustomRules(customRulesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom rules: %v", err)
	}

	return &Analyzer{
		filePath:       filePath,
		customRules:    customRules,
		modules:        make(map[string]*Analyzer),
		issues:         []Issue{},
		variables:      make(map[string]bool),
		usedVariables:  make(map[string]bool),
		definedModules: make(map[string]bool),
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

	a.analyzeModule()
	a.analyzeSubmodules()
	a.analyzeVaiableUsage()

	fmt.Println("Analyzing Terraform config...", a.filePath)

	return nil
}

func (a *Analyzer) analyzeModule() {
	// analyze the current module
	a.checkUnusedVariables()
	a.checkMissingRequiredAttributes()
	a.checkDuplicateResourceNames()
	a.applyCustomRules()
	a.collectVariables()
}

func (a *Analyzer) analyzeSubmodules() {
	for _, block := range a.body.Blocks {
		if block.Type == "module" {
			moduleName := block.Labels[0]
			modulePath := filepath.Join(filepath.Dir(a.filePath), block.Labels[1])
			submoduleAnalyzer, err := NewAnalyzer(modulePath, "")
			if err != nil {
				a.issues = append(a.issues, Issue{
					Module:  moduleName,
					Message: fmt.Sprintf("Failed to analyze submodule: %v", err),
				})
				continue
			}
			err = submoduleAnalyzer.Analyze()
			if err != nil {
				a.issues = append(a.issues, Issue{
					Module:  moduleName,
					Message: fmt.Sprintf("Failed to analyze submodule: %v", err),
				})
				continue
			}

			a.modules[moduleName] = submoduleAnalyzer
			a.issues = append(a.issues, submoduleAnalyzer.issues...)
		}

		for _, submoduleAnalyzer := range a.modules {
			a.usedVariables = mergeStringBoolMaps(a.usedVariables, submoduleAnalyzer.usedVariables)
			a.definedModules = mergeStringBoolMaps(a.definedModules, submoduleAnalyzer.definedModules)
	}
}

func (a *Analyzer) collectVariables() {
	for _, block := range a.body.Blocks {
		if block.Type == "variable" {
			variableName := block.Labels[0]
			a.variables[variableName] = true
		}
	}
}

func (a *Analyzer) analyzeVariableUsage() {
	for _, attribute := range a.body.Attributes {
		vars := attribute.Expr.Variables()
		for _, v := range vars {
			a.usedVariables[v.RootName()] = true
		}
	}

	for _, block := range a.body.Blocks {
		if block.Type == "module" {
			moduleName := block.Labels[0]
			a.definedModules[moduleName] = true
		}
	}

	for variable := range a.variables {
		if !a.usedVariables[variable] {
			a.issues = append(a.issues, Issue{
				Module:  a.filePath,
				Message: fmt.Sprintf("Unused variable: %s", variable),
			})
		}
	}

	for usedVariable := range a.usedVariables {
		if !a.variables[usedVariable] && !a.definedModules[usedVariable] {
			a.issues = append(a.issues, Issue{
				Module:  a.filePath,
				Message: fmt.Sprintf("Used but undefined variable: %s", usedVariable),
			})
		}
	}
}

func mergeStringBoolMaps(map1, map2 map[string]bool) map[string]bool {
	mergedMap := make(map[string]bool)
	for k, v := range map1 {
		mergedMap[k] = v
	}
	for k, v := range map2 {
		mergedMap[k] = v
	}
	return mergedMap
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

func (a *Analyzer) PrintReport() {
	fmt.Printf("Analysis report for %s\n", a.filePath)
	for _, issue := range a.issues {
		fmt.Printf("Module: %s - %s\n", issue.Module, issue.Message)
	}
}
