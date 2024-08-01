package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/justisGipson/tf_analyzer/pkg/analyzer"
)

func main() {
	filePath := flag.String("file", "", "Path to the Terraform configuration file")
	customRulesPath := flag.String("rules", "", "Path to the custom rules file")
	flag.Parse()
	if *filePath == "" || *customRulesPath == "" {
		fmt.Println("Please provide a Terraform configuration file path using the -file flag")
		os.Exit(1)
	}

	analyzer, err := analyzer.NewAnalyzer(*filePath, *customRulesPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	err = analyzer.Analyze()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
