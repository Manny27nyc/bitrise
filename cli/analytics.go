package cli

import (
	"bytes"
	"fmt"
	"io"

	"github.com/bitrise-io/bitrise/tools/filterwriter"
	"github.com/bitrise-io/envman/env"
	"github.com/bitrise-io/envman/models"
)

func expandStepInputsForAnalytics(environment, inputs []models.EnvironmentItemModel, secrets []string) (map[string]string, error) {
	for _, newEnv := range environment {
		if err := newEnv.FillMissingDefaults(); err != nil {
			return map[string]string{}, fmt.Errorf("could not fill missing environment model (%s) defaults: %s", newEnv, err)
		}
	}

	sideEffects, err := env.GetDeclarationsSideEffects(environment, &env.DefaultEnvironmentSource{})
	if err != nil {
		return map[string]string{}, fmt.Errorf("getting step environment declaration results failed: %s", err)
	}

	// Filter inputs from enviroments
	expandedInputs := make(map[string]string)
	for _, input := range inputs {
		inputKey, _, err := input.GetKeyValuePair()
		if err != nil {
			return map[string]string{}, fmt.Errorf("failed to get input key: %s", err)
		}

		// If input key may not be present in the result environment.
		// This can happen if the input has the "skip_if_empty" property set to true, and it is empty.
		inputValue, ok := sideEffects.ResultEnvironment[inputKey]
		if !ok {
			expandedInputs[inputKey] = ""
			continue
		}

		src := bytes.NewReader([]byte(inputValue))
		dstBuf := new(bytes.Buffer)
		secretFilterDst := filterwriter.New(secrets, dstBuf)

		if _, err := io.Copy(secretFilterDst, src); err != nil {
			return map[string]string{}, fmt.Errorf("failed to redact secrets, stream copy failed: %s", err)
		}
		if _, err := secretFilterDst.Flush(); err != nil {
			return map[string]string{}, fmt.Errorf("failed to redact secrets, stream flush failed: %s", err)
		}

		expandedInputs[inputKey] = dstBuf.String()
	}

	return expandedInputs, nil
}
