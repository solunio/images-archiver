package main

import (
	"fmt"

	"go.podman.io/image/v5/signature"
)

// createPolicyContext creates a signature policy context that accepts anything
func createPolicyContext() (*signature.PolicyContext, error) {
	policy := &signature.Policy{
		Default: []signature.PolicyRequirement{
			signature.NewPRInsecureAcceptAnything(),
		},
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, fmt.Errorf("error creating policy context: %w", err)
	}
	return policyContext, nil
}
