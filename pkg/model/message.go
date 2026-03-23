package model

type DocumentParseType int

const (
	DocumentParseTypeQA DocumentParseType = iota // 问答类型
	DocumentParseTypeSegment
	DocumentParseTypeGraph
)

// DocumentParseRequest DTO 定义了触发文档解析任务所需的信息
/*
{
  "docJobId": "int64", // 整体接入任务ID，必填
  "docId": "int64", // 文档在数据库中的唯一ID，必填
  "fileName": "string", // 文档名称，必填
  "filePath": "string" // 文档在对象存储中的URI，必填
  "contentType": "string", // 文档类型，必填
  "libId": "int64", // 知识库ID，必填
}
*/
type DocumentParseRequest struct {
	JobID           int64  `json:"docJobId"` // 整体接入任务 ID
	DocumentID      int64  `json:"docId"`    // 文档在数据库中的唯一 ID
	DocumentName    string `json:"fileName"` // 文档名称
	URI             string `json:"filePath"`
	KnowledgeBaseId int64  `json:"libId"`          // 知识库 ID
	Type            string `json:"contentType"`    // 文档类型
	OfficeFileName  string `json:"officeFileName"` // OnlyOffice预览文件名
	Size            int64  `json:"size"`           // 文件大小
}

type DocumentChunkEmbedRequest struct {
	JobID           int64  `json:"chunkJobId"` // 任务ID
	ChunkID         int64  `json:"chunkId"`    // 块ID
	ParentJobID     int64  `json:"docJobId"`   // 整体接入任务ID
	DocumentID      int64  `json:"docId"`      // 文档ID
	DocumentName    string `json:"fileName"`   // 文档名称
	KnowledgeBaseId int64  `json:"libId"`      // 知识库ID
	ChunkIndex      int    `json:"chunkIndex"` // 块序号
	ChunkName       string `json:"chunkName"`  // 块名称
	ChunkURI        string `json:"chunkUri"`   // 块在对象存储中的URI
	SourceType      string `json:"sourceType"` // 文档类型
}

type DocumentDeepLearnRequest struct {
	JobID           int64  `json:"jobId"`           // 任务ID
	DocumentID      int64  `json:"documentId"`      // 文档ID
	DocumentName    string `json:"documentName"`    // 文档名称
	KnowledgeBaseId int64  `json:"knowledgeBaseId"` // 知识库ID
}
type DocumentDeepThinkRequest struct {
	JobID           int64  `json:"jobId"`           // 整体接入任务 ID
	DocumentID      int64  `json:"documentId"`      // 文档ID
	DocumentName    string `json:"documentName"`    // 文档名称
	KnowledgeBaseId int64  `json:"knowledgeBaseId"` // 知识库ID
}
