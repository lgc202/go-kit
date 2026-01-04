package embeddings

import (
	"encoding/json"
	"fmt"
)

type embeddingRequest struct {
	provider string `json:"-"`

	Model string   `json:"model"`
	Input []string `json:"input"`

	User *string `json:"user,omitempty"`

	extra                   map[string]any `json:"-"`
	allowExtraFieldOverride bool           `json:"-"`
}

func (r embeddingRequest) MarshalJSON() ([]byte, error) {
	type alias embeddingRequest
	base, err := json.Marshal(alias(r))
	if err != nil {
		return nil, err
	}
	if len(r.extra) == 0 {
		return base, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(base, &obj); err != nil {
		return nil, err
	}

	for k, v := range r.extra {
		if !r.allowExtraFieldOverride {
			if _, exists := obj[k]; exists {
				return nil, fmt.Errorf("%s: extra field %q conflicts with a built-in option (set llm.WithAllowExtraFieldOverride(true) to override)", r.provider, k)
			}
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		obj[k] = b
	}

	return json.Marshal(obj)
}
