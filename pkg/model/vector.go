package model

import "time"

type ParagraphCollection struct {
	ID              int64       `json:"id"`
	ChunkIndex      int         `json:"chunkIndex"`
	DocumentID      int64       `json:"documentId"`
	KnowledgeBaseID int64       `json:"knowledgeBaseId"`
	Text            string      `json:"text"`
	Order           int         `json:"order"`
	Embedding       [][]float32 `json:"embedding"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}

type QACollection struct {
	ID                int64       `json:"id"`
	ChunkIndex        int         `json:"chunkIndex"`
	DocumentID        int64       `json:"documentId"`
	KnowledgeBaseID   int64       `json:"knowledgeBaseId"`
	Question          string      `json:"question"`
	Answer            string      `json:"answer"`
	QuestionEmbedding [][]float32 `json:"questionEmbedding"`
	AnswerEmbedding   [][]float32 `json:"answerEmbedding"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`
}

type KnowledgeCollection struct {
	ID              int64       `json:"id"`
	ChunkIndex      int         `json:"chunkIndex"`
	DocumentID      int64       `json:"documentId"`
	KnowledgeBaseID int64       `json:"knowledgeBaseId"`
	Embedding       [][]float32 `json:"embedding"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}
