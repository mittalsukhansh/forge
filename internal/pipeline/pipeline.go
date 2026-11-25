package pipeline

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Step struct {
	Name     string   `yaml:"name" json:"name"`
	Image    string   `yaml:"image" json:"image"`
	Commands []string `yaml:"commands" json:"commands"`
}

type Pipeline struct {
	Name  string `yaml:"name" json:"name"`
	Steps []Step `yaml:"steps" json:"steps"`
}

func Parse(data []byte) (*Pipeline, error) {
	var p Pipeline

	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return &p, nil
}

func Load(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return Parse(data)
}

func (p *Pipeline) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("pipeline name is required")
	}

	if len(p.Steps) == 0 {
		return fmt.Errorf("pipeline must contain at least one step")
	}

	for i, step := range p.Steps {
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("step %d: name is required", i+1)
		}

		if strings.TrimSpace(step.Image) == "" {
			return fmt.Errorf("step %d: image is required", i+1)
		}

		if strings.HasPrefix(step.Image, "-") {
			return fmt.Errorf("step %d: invalid image", i+1)
		}

		if len(step.Commands) == 0 {
			return fmt.Errorf("step %d: commands are required", i+1)
		}
	}

	return nil
}
