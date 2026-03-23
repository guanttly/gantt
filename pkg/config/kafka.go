package config

import "time"

type KafkaConfig struct {
	Producer *KafkaProducerConfig   `mapstructure:"producer" yaml:"producer"` // Kafka 生产者配置
	Consumer []*KafkaConsumerConfig `mapstructure:"consumer" yaml:"consumer"` // Kafka 消费者配置
}

// KafkaProducerConfig 用于配置 Kafka 生产者
type KafkaProducerConfig struct {
	Brokers               []string      `mapstructure:"brokers" yaml:"brokers"`                                   // Kafka Broker 地址列表
	ParseTaskTopic        string        `mapstructure:"parse_task_topic" yaml:"parse_task_topic"`                 // 解析请求主题名称
	ChunkEmbedTaskTopic   string        `mapstructure:"chunk_embed_task_topic" yaml:"chunk_embed_task_topic"`     // 文档分块嵌入任务主题名称
	ReportChunkParseTopic string        `mapstructure:"report_chunk_parse_topic" yaml:"report_chunk_parse_topic"` // 文档分块嵌入结果报告主题名称
	ReportJobFailTopic    string        `mapstructure:"report_job_fail_topic" yaml:"report_job_fail_topic"`       // 任务失败报告主题名称
	DeepLearningTaskTopic string        `mapstructure:"deep_learning_task_topic" yaml:"deep_learning_task_topic"` // 深度学习任务主题名称
	DeepThinkingTaskTopic string        `mapstructure:"deep_thinking_task_topic" yaml:"deep_thinking_task_topic"` // 深度思考任务主题名称
	WriteTimeout          time.Duration `mapstructure:"write_timeout" yaml:"write_timeout"`                       // 写入超时，例如 10*time.Second
	RequiredAcks          int           `mapstructure:"required_acks" yaml:"required_acks"`                       // 例如 kafka.RequireOne, kafka.RequireAll
}

type KafkaConsumerConfig struct {
	Brokers        []string      `mapstructure:"brokers" yaml:"brokers"`                 // Kafka Broker 地址列表
	Topic          string        `mapstructure:"topic" yaml:"topic"`                     // 消费主题名称
	GroupID        string        `mapstructure:"group_id" yaml:"group_id"`               // 消费者组 ID
	StartOffset    int64         `mapstructure:"start_offset" yaml:"start_offset"`       // e.g., kafka.FirstOffset, kafka.LastOffset. Default is LastOffset.
	MinBytes       int           `mapstructure:"min_bytes" yaml:"min_bytes"`             // Default: 1
	MaxBytes       int           `mapstructure:"max_bytes" yaml:"max_bytes"`             // Default: 1MB
	MaxWait        time.Duration `mapstructure:"max_wait" yaml:"max_wait"`               // Default: 10s
	CommitInterval time.Duration `mapstructure:"commit_interval" yaml:"commit_interval"` // Auto-commit interval if not using manual commit. 0 for manual. Default: 0
}
