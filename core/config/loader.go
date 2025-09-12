// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"github.com/pelletier/go-toml/v2"
)

// LoadConfig loads configuration from a file, automatically trying different formats
// if the specified file doesn't exist
func LoadConfig(filePath string) (*Config, error) {
	// First try the exact path provided
	if _, err := os.Stat(filePath); err == nil {
		return loadConfigFile(filePath)
	}

	// If file doesn't exist, try different extensions
	basePath := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	extensions := []string{".json", ".toml", ".yaml", ".yml"}
	
	for _, ext := range extensions {
		tryPath := basePath + ext
		if _, err := os.Stat(tryPath); err == nil {
			return loadConfigFile(tryPath)
		}
	}
	
	// If we get here, no suitable file was found
	return nil, fmt.Errorf("config file not found: %s (tried json, toml, yaml, yml formats)", filePath)
}

// loadConfigFile loads a specific config file based on its extension
func loadConfigFile(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	// For non-JSON formats, first convert to a generic map, then to JSON
	if ext != ".json" {
		var genericMap map[string]interface{}
		
		switch ext {
		case ".toml":
			if err := toml.Unmarshal(data, &genericMap); err != nil {
				return nil, fmt.Errorf("error parsing TOML: %w", err)
			}
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, &genericMap); err != nil {
				return nil, fmt.Errorf("error parsing YAML: %w", err)
			}
		}
		
		// Convert the generic map to JSON bytes
		data, err = json.Marshal(genericMap)
		if err != nil {
			return nil, fmt.Errorf("error converting to JSON: %w", err)
		}
	}

	// Now parse the JSON (either original or converted)
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	fmt.Printf("Loaded configuration from %s\n", filePath)
	return &config, nil
}

// ConvertToJSON converts the config to JSON bytes with indentation
func ConvertToJSON(config *Config) ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// SaveAsJSON saves the config as a JSON file
func SaveAsJSON(config *Config, filePath string) error {
	jsonData, err := ConvertToJSON(config)
	if err != nil {
		return fmt.Errorf("error converting config to JSON: %w", err)
	}
	
	if err := ioutil.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing JSON file: %w", err)
	}
	
	fmt.Printf("Configuration saved to %s\n", filePath)
	return nil
}

// ConvertToYAML converts the config to YAML bytes
func ConvertToYAML(config *Config) ([]byte, error) {
	// First convert to JSON
	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("error converting config to JSON: %w", err)
	}
	
	// Then convert JSON to a generic map
	var genericMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &genericMap); err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}
	
	// Finally convert the map to YAML
	return yaml.Marshal(genericMap)
}

// SaveAsYAML saves the config as a YAML file
func SaveAsYAML(config *Config, filePath string) error {
	yamlData, err := ConvertToYAML(config)
	if err != nil {
		return fmt.Errorf("error converting config to YAML: %w", err)
	}
	
	if err := ioutil.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("error writing YAML file: %w", err)
	}
	
	fmt.Printf("Configuration saved to %s\n", filePath)
	return nil
}
