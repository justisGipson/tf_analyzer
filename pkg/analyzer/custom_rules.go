package analyzer

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type CustomRule struct {
	ResourceType string   `yaml:"resource_type"`
	Attributes   []string `yaml:"attributes"`
	Validator    []string `yaml:"validator"`
}

type CustomRules struct {
	Rules []CustomRule `yaml:"rules"`
}

func LoadCustomRules(filePath string) ([]CustomRule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom rules file: %w", err)
	}

	var customRules CustomRules
	err = yaml.Unmarshal(data, &customRules)
	if err != nil {
		return nil, fmt.Errorf("failed to parse custom rules file: %w", err)
	}

	return customRules.Rules, nil
}
