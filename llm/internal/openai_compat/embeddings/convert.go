package embeddings

import "github.com/lgc202/go-kit/llm/schema"

func toSchemaEmbeddingResponse(in embeddingResponse) schema.EmbeddingResponse {
	out := schema.EmbeddingResponse{
		Model: in.Model,
		Usage: schema.Usage{
			PromptTokens: in.Usage.PromptTokens,
			TotalTokens:  in.Usage.TotalTokens,
		},
	}

	out.Data = make([]schema.Embedding, 0, len(in.Data))
	for _, d := range in.Data {
		out.Data = append(out.Data, schema.Embedding{
			Index:  d.Index,
			Vector: d.Embedding,
		})
	}

	return out
}
