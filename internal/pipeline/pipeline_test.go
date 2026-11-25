package pipeline

import "testing"

func TestValidPipeline(t *testing.T) {
	data := []byte(`
name: Test Pipeline
steps:
  - name: Test
    image: alpine:3
    commands:
      - echo hello
`)

	p, err := Parse(data)

	if err != nil {
		t.Fatal(err)
	}

	if p.Name != "Test Pipeline" {
		t.Fatal("Incorrect pipeline name")
	}

	if len(p.Steps) != 1 {
		t.Fatal("Incorrect step count")
	}
}

func TestInvalidPipelines(t *testing.T) {
	tests := map[string]string{
		"Invalid YAML":  "name: [",
		"Missing Name":  "steps: []",
		"Missing Steps": "name: Test",
		"Missing Step Name": `
name: Test
steps:
  - image: alpine:3
    commands: [echo hello]
`,
		"Missing Image": `
name: Test
steps:
  - name: Test
    commands: [echo hello]
`,
		"Missing Commands": `
name: Test
steps:
  - name: Test
    image: alpine:3
`,
		"Invalid Image": `
name: Test
steps:
  - name: Test
    image: --privileged
    commands: [echo hello]
`,
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse([]byte(data)); err == nil {
				t.Fatal("Expected validation error")
			}
		})
	}
}
