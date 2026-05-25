package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/kilip/sbctl/internal/config"
)

func main() {
	s := jsonschema.Reflect(&config.Config{})
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal schema: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("docs/config-schema.json", data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write schema file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("JSON Schema generated successfully at docs/config-schema.json")
}
