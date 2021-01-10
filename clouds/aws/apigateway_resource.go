package aws

import (
	"encoding/json"

	"github.com/operatorai/operator/command"
	"github.com/operatorai/operator/config"
)

func setRestApiResourceID(cfg *config.TemplateConfig) error {
	if cfg.RestApiResourceID != "" {
		return nil
	}

	// Look for existing resource ID
	resourceID, _, err := getRestApiResource(cfg)
	if err != nil {
		return err
	}
	if resourceID != "" {
		cfg.RestApiResourceID = resourceID
		return nil
	}

	// Create a resource in the API
	// s := spinner.StartNew("Creating an AWS API gateway resource...")
	// defer s.Stop()
	output, err := command.ExecuteWithResult("aws", []string{
		"apigateway",
		"create-resource",
		"--rest-api-id", cfg.RestApiID,
		"--path-part", cfg.Name,
		"--parent-id", cfg.RestApiRootID,
	})
	if err != nil {
		return err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return err
	}
	cfg.RestApiResourceID = result.ID
	return nil
}

func getRestApiResource(cfg *config.TemplateConfig) (string, bool, error) {
	// fmt.Println("Collecting AWS API resources...")
	// s := spinner.StartNew("Querying...")
	// defer s.Stop()

	output, err := command.ExecuteWithResult("aws", []string{
		"apigateway",
		"get-resources",
		"--rest-api-id", cfg.RestApiID,
	})
	if err != nil {
		return "", false, err
	}

	var results struct {
		Items []struct {
			PathPart        string `json:"pathPart"`
			ID              string `json:"id"`
			ResourceMethods struct {
				POST *struct{} `json:"POST"`
			} `json:"resourceMethods"`
		} `json:"items"`
	}
	if err := json.Unmarshal(output, &results); err != nil {
		return "", false, err
	}

	for _, result := range results.Items {
		if result.PathPart == cfg.Name {
			return result.ID, (result.ResourceMethods.POST != nil), nil
		}
	}
	return "", false, nil
}

func createRestApiResourceMethod(cfg *config.TemplateConfig) error {
	_, resourceHasPOSTMethod, err := getRestApiResource(cfg)
	if err != nil {
		return err
	}
	if resourceHasPOSTMethod {
		return nil
	}

	// fmt.Println("Creating POST method in API resource...")
	// s := spinner.StartNew("Querying...")
	// defer s.Stop()

	// Create the method
	err = command.Execute("aws", []string{
		"apigateway",
		"put-method",
		"--rest-api-id", cfg.RestApiID,
		"--resource-id", cfg.RestApiResourceID,
		"--http-method", "POST",
		"--authorization-type", "NONE",
	}, true)
	if err != nil {
		return err
	}

	// Set the method response to JSON
	err = command.Execute("aws", []string{
		"apigateway",
		"put-method-response",
		"--rest-api-id", cfg.RestApiID,
		"--resource-id", cfg.RestApiResourceID,
		"--http-method", "POST",
		"--status-code", "200",
		"--response-models", "application/json=Empty",
	}, true)
	if err != nil {
		return err
	}
	return nil
}