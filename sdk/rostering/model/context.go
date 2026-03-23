package model

// MemoryRecord 内存记录
type MemoryRecord struct {
	ID    uint64            `json:"id,omitempty"`
	Key   string            `json:"key"`
	Value string            `json:"value"`
	Tags  []string          `json:"tags,omitempty"`
	Meta  map[string]string `json:"meta,omitempty"`
}

// MemorySearchResult 语义搜索结果
type MemorySearchResult struct {
	Memory MemoryRecord `json:"memory"`
	Score  float64      `json:"score"`
}

// 注意：ConversationRecord 和 ConversationMessage 已迁移到 jusha/agent/sdk/context/model
// 如需使用，请导入 jusha/agent/sdk/context/model 包

// Memory Create
type MemoryCreateRequest struct {
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Tags      []string          `json:"tags,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	AutoEmbed bool              `json:"auto_embed,omitempty"`
}

type MemoryCreateResponse struct {
	Memory    MemoryRecord   `json:"memory,omitempty"`
	Embedding map[string]any `json:"embedding,omitempty"`
}

// Memory Read
type MemoryReadRequest struct {
	ID uint64 `json:"id"`
}

type MemoryReadResponse struct {
	MemoryRecord
}

// Memory Update
type MemoryUpdateRequest struct {
	ID    uint64            `json:"id"`
	Key   string            `json:"key,omitempty"`
	Value string            `json:"value,omitempty"`
	Tags  []string          `json:"tags,omitempty"`
	Meta  map[string]string `json:"meta,omitempty"`
}

type MemoryUpdateResponse struct {
	MemoryRecord
}

// Memory Delete
type MemoryDeleteRequest struct {
	ID uint64 `json:"id"`
}

// Memory Search
type MemorySearchRequest struct {
	Tags  []string `json:"tags"`
	Limit int      `json:"limit,omitempty"`
}

type MemorySearchResponse struct {
	Memories []MemoryRecord `json:"memories"`
}

// Memory Semantic Search
type MemorySemanticSearchRequest struct {
	Query  string         `json:"query"`
	TopK   int            `json:"top_k,omitempty"`
	Filter map[string]any `json:"filter,omitempty"`
}

type MemorySemanticSearchResponse struct {
	Results []MemorySearchResult `json:"results"`
}

// Memory Embedding Upsert
type MemoryEmbeddingUpsertRequest struct {
	ID uint64 `json:"id"`
}

type MemoryEmbeddingUpsertResponse struct {
	Status string `json:"status"`
	ID     uint64 `json:"id"`
}

// Memory Bulk Embedding Upsert
type MemoryBulkEmbeddingUpsertRequest struct {
	IDs         []uint64 `json:"ids,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
	Retry       int      `json:"retry,omitempty"`
}

type MemoryBulkEmbeddingUpsertResponse struct {
	Total         int               `json:"total"`
	SuccessIDs    []uint64          `json:"success_ids"`
	FailedIDs     []uint64          `json:"failed_ids"`
	FailedDetails map[string]string `json:"failed_details,omitempty"`
	FromTags      bool              `json:"from_tags"`
	Metrics       map[string]any    `json:"metrics,omitempty"`
}

// Memory Build Context
type MemoryBuildContextRequest struct {
	ConversationID string `json:"conversation_id"`
	Query          string `json:"query"`
	LimitHistory   int    `json:"limit_history,omitempty"`
	TopK           int    `json:"top_k,omitempty"`
}

type MemoryBuildContextResponse struct {
	ConversationID  string               `json:"conversation_id"`
	Query           string               `json:"query"`
	History         []map[string]any      `json:"history"` // 使用通用类型，避免依赖 context model
	RelatedMemories []MemorySearchResult `json:"related_memories"`
}

// 注意：Conversation 相关的模型已迁移到 jusha/agent/sdk/context/model
// 请使用 jusha/agent/sdk/context/model 包中的类型
