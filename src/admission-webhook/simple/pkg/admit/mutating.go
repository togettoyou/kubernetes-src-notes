package admit

import (
	"encoding/json"

	admissionv1 "k8s.io/api/admission/v1"
)

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func PatchTypeJSONPatch(patch ...PatchOperation) ([]byte, *admissionv1.PatchType, error) {
	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		return nil, nil, err
	}
	pt := admissionv1.PatchTypeJSONPatch
	return patchBytes, &pt, nil
}
