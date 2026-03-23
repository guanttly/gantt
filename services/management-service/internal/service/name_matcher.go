package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/mcp/pkg/ai"
	"jusha/mcp/pkg/logging"
)

// NameMatcher 名称模糊匹配器
// 用于将 LLM 返回的班次/员工名称匹配为实际的 ID
// 支持多层匹配策略：精确匹配 → 核心名称匹配 → 模糊匹配 → LLM 智能匹配
type NameMatcher struct {
	logger       logging.ILogger
	aiFactory    *ai.AIProviderFactory
	employeeRepo repository.IEmployeeRepository
	shiftRepo    repository.IShiftRepository
	groupRepo    repository.IGroupRepository
}

// NewNameMatcher 创建名称匹配器
func NewNameMatcher(
	logger logging.ILogger,
	aiFactory *ai.AIProviderFactory,
	employeeRepo repository.IEmployeeRepository,
	shiftRepo repository.IShiftRepository,
	groupRepo repository.IGroupRepository,
) *NameMatcher {
	return &NameMatcher{
		logger:       logger,
		aiFactory:    aiFactory,
		employeeRepo: employeeRepo,
		shiftRepo:    shiftRepo,
		groupRepo:    groupRepo,
	}
}

// MatchEmployeeName 匹配员工名称
// 使用多层匹配策略：精确匹配 → 核心名称匹配 → 模糊匹配 → LLM 智能匹配
func (m *NameMatcher) MatchEmployeeName(ctx context.Context, orgID, name string) (string, error) {
	// 获取所有员工（不使用 Keyword 过滤，以便进行全量匹配）
	employees, err := m.employeeRepo.List(ctx, &model.EmployeeFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("查询员工失败: %w", err)
	}

	if employees == nil || len(employees.Items) == 0 {
		return "", fmt.Errorf("未找到任何员工")
	}

	// 构建候选列表
	candidates := make([]matchCandidate, 0, len(employees.Items))
	for _, emp := range employees.Items {
		candidates = append(candidates, matchCandidate{
			ID:   emp.ID,
			Name: emp.Name,
		})
	}

	// 多层匹配
	matchedID, confidence, matchType := m.multiLayerMatch(name, candidates)
	if matchedID != "" {
		m.logger.Info("员工名称匹配成功",
			"input", name,
			"matchedID", matchedID,
			"confidence", confidence,
			"matchType", matchType)
		return matchedID, nil
	}

	// 传统匹配失败，尝试 LLM 智能匹配
	if m.aiFactory != nil {
		matchedID, err = m.llmMatch(ctx, name, candidates, "employee")
		if err == nil && matchedID != "" {
			m.logger.Info("员工名称 LLM 匹配成功", "input", name, "matchedID", matchedID)
			return matchedID, nil
		}
		m.logger.Warn("员工名称 LLM 匹配失败", "input", name, "error", err)
	}

	return "", fmt.Errorf("未找到匹配的员工: %s", name)
}

// MatchShiftName 匹配班次名称
// 使用多层匹配策略：精确匹配 → 核心名称匹配 → 模糊匹配 → LLM 智能匹配
func (m *NameMatcher) MatchShiftName(ctx context.Context, orgID, name string) (string, error) {
	// 获取所有班次（不使用 Keyword 过滤，以便进行全量匹配）
	shifts, err := m.shiftRepo.List(ctx, &model.ShiftFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("查询班次失败: %w", err)
	}

	if shifts == nil || len(shifts.Items) == 0 {
		return "", fmt.Errorf("未找到任何班次")
	}

	// 构建候选列表
	candidates := make([]matchCandidate, 0, len(shifts.Items))
	for _, shift := range shifts.Items {
		candidates = append(candidates, matchCandidate{
			ID:   shift.ID,
			Name: shift.Name,
		})
	}

	// 多层匹配
	matchedID, confidence, matchType := m.multiLayerMatch(name, candidates)
	if matchedID != "" {
		m.logger.Info("班次名称匹配成功",
			"input", name,
			"matchedID", matchedID,
			"confidence", confidence,
			"matchType", matchType)
		return matchedID, nil
	}

	// 传统匹配失败，尝试 LLM 智能匹配
	if m.aiFactory != nil {
		matchedID, err = m.llmMatch(ctx, name, candidates, "shift")
		if err == nil && matchedID != "" {
			m.logger.Info("班次名称 LLM 匹配成功", "input", name, "matchedID", matchedID)
			return matchedID, nil
		}
		m.logger.Warn("班次名称 LLM 匹配失败", "input", name, "error", err)
	}

	return "", fmt.Errorf("未找到匹配的班次: %s", name)
}

// MatchGroupName 匹配分组名称
// 使用多层匹配策略：精确匹配 → 核心名称匹配 → 模糊匹配 → LLM 智能匹配
func (m *NameMatcher) MatchGroupName(ctx context.Context, orgID, name string) (string, error) {
	groups, err := m.groupRepo.List(ctx, &model.GroupFilter{
		OrgID:    orgID,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		return "", fmt.Errorf("查询分组失败: %w", err)
	}

	if groups == nil || len(groups.Items) == 0 {
		return "", fmt.Errorf("未找到任何分组")
	}

	// 构建候选列表
	candidates := make([]matchCandidate, 0, len(groups.Items))
	for _, group := range groups.Items {
		candidates = append(candidates, matchCandidate{
			ID:   group.ID,
			Name: group.Name,
		})
	}

	// 多层匹配
	matchedID, confidence, matchType := m.multiLayerMatch(name, candidates)
	if matchedID != "" {
		m.logger.Info("分组名称匹配成功",
			"input", name,
			"matchedID", matchedID,
			"confidence", confidence,
			"matchType", matchType)
		return matchedID, nil
	}

	// 传统匹配失败，尝试 LLM 智能匹配
	if m.aiFactory != nil {
		matchedID, err = m.llmMatch(ctx, name, candidates, "group")
		if err == nil && matchedID != "" {
			m.logger.Info("分组名称 LLM 匹配成功", "input", name, "matchedID", matchedID)
			return matchedID, nil
		}
		m.logger.Warn("分组名称 LLM 匹配失败", "input", name, "error", err)
	}

	return "", fmt.Errorf("未找到匹配的分组: %s", name)
}

// matchCandidate 匹配候选项
type matchCandidate struct {
	ID   string
	Name string
}

// multiLayerMatch 多层匹配策略
// 返回: matchedID, confidence, matchType
func (m *NameMatcher) multiLayerMatch(input string, candidates []matchCandidate) (string, float64, string) {
	inputNorm := normalizeChineseName(input)
	inputCore := extractCoreName(input)

	// 第一层：精确匹配
	for _, c := range candidates {
		if c.Name == input {
			return c.ID, 1.0, "exact"
		}
	}

	// 第二层：标准化后精确匹配
	for _, c := range candidates {
		if normalizeChineseName(c.Name) == inputNorm {
			return c.ID, 0.98, "normalized_exact"
		}
	}

	// 第三层：核心名称匹配（去除括号后缀）
	// 例如 "上夜班（江北）" → "上夜班"
	if inputCore != input {
		for _, c := range candidates {
			candidateCore := extractCoreName(c.Name)
			if candidateCore == inputCore {
				return c.ID, 0.95, "core_exact"
			}
		}
	}

	// 第四层：双向包含匹配
	// 输入包含候选，或候选包含输入
	for _, c := range candidates {
		candidateNorm := normalizeChineseName(c.Name)
		if strings.Contains(inputNorm, candidateNorm) || strings.Contains(candidateNorm, inputNorm) {
			return c.ID, 0.85, "contains"
		}
	}

	// 第五层：核心名称包含匹配
	for _, c := range candidates {
		candidateCore := extractCoreName(c.Name)
		if strings.Contains(inputCore, candidateCore) || strings.Contains(candidateCore, inputCore) {
			return c.ID, 0.80, "core_contains"
		}
	}

	// 第六层：相似度匹配（Levenshtein + 中文优化）
	bestMatch := ""
	bestScore := 0.0
	for _, c := range candidates {
		// 同时计算原始名称和核心名称的相似度
		score1 := m.calculateChineseSimilarity(inputNorm, normalizeChineseName(c.Name))
		score2 := m.calculateChineseSimilarity(inputCore, extractCoreName(c.Name))
		score := max(score1, score2)

		if score > bestScore {
			bestScore = score
			bestMatch = c.ID
		}
	}

	if bestMatch != "" && bestScore >= 0.6 {
		return bestMatch, bestScore, "similarity"
	}

	return "", 0, ""
}

// normalizeChineseName 标准化中文名称
// 去除空格、统一括号、统一大小写
func normalizeChineseName(name string) string {
	// 去除首尾空格
	name = strings.TrimSpace(name)

	// 统一中英文括号
	name = strings.ReplaceAll(name, "（", "(")
	name = strings.ReplaceAll(name, "）", ")")
	name = strings.ReplaceAll(name, "【", "[")
	name = strings.ReplaceAll(name, "】", "]")

	// 去除多余空格
	name = strings.Join(strings.Fields(name), "")

	return name
}

// extractCoreName 提取核心名称（去除括号及其内容）
// 例如 "上夜班（江北）" → "上夜班"
// 例如 "早班[本部]" → "早班"
func extractCoreName(name string) string {
	// 去除各种括号及其内容
	patterns := []string{
		`\([^)]*\)`,  // 英文圆括号
		`（[^）]*）`,    // 中文圆括号
		`\[[^\]]*\]`, // 英文方括号
		`【[^】]*】`,    // 中文方括号
		`-[^-]+$`,    // 末尾的连字符后缀
		`_[^_]+$`,    // 末尾的下划线后缀
	}

	result := name
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	return strings.TrimSpace(result)
}

// calculateChineseSimilarity 计算中文相似度
// 结合 Levenshtein 距离和字符重叠度
func (m *NameMatcher) calculateChineseSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// 转换为 rune 切片处理中文
	r1, r2 := []rune(s1), []rune(s2)
	if len(r1) == 0 || len(r2) == 0 {
		return 0.0
	}

	// 计算字符重叠度（Jaccard 相似度）
	set1 := make(map[rune]bool)
	set2 := make(map[rune]bool)
	for _, r := range r1 {
		if !unicode.IsSpace(r) {
			set1[r] = true
		}
	}
	for _, r := range r2 {
		if !unicode.IsSpace(r) {
			set2[r] = true
		}
	}

	intersection := 0
	for r := range set1 {
		if set2[r] {
			intersection++
		}
	}
	union := len(set1) + len(set2) - intersection
	jaccard := 0.0
	if union > 0 {
		jaccard = float64(intersection) / float64(union)
	}

	// 计算 Levenshtein 相似度
	distance := m.levenshteinDistance(s1, s2)
	maxLen := len(r1)
	if len(r2) > maxLen {
		maxLen = len(r2)
	}
	levenshtein := 1.0 - float64(distance)/float64(maxLen)

	// 综合评分：70% Levenshtein + 30% Jaccard
	return 0.7*levenshtein + 0.3*jaccard
}

// llmMatch 使用 LLM 进行智能匹配
func (m *NameMatcher) llmMatch(ctx context.Context, input string, candidates []matchCandidate, entityType string) (string, error) {
	if len(candidates) == 0 {
		return "", fmt.Errorf("候选列表为空")
	}

	// 构建候选名称列表
	candidateNames := make([]string, 0, len(candidates))
	nameToID := make(map[string]string)
	for _, c := range candidates {
		candidateNames = append(candidateNames, c.Name)
		nameToID[c.Name] = c.ID
	}

	entityTypeCN := map[string]string{
		"shift":    "班次",
		"employee": "员工",
		"group":    "分组",
	}[entityType]

	systemPrompt := fmt.Sprintf(`你是一个%s名称匹配专家。用户会提供一个输入名称和一个候选列表，你需要从候选列表中选择最匹配的项。

规则：
1. 如果输入名称是简写，应匹配完整名称
2. 如果有多个可能的匹配，选择最接近的一个
3. 如果没有合适的匹配，返回 null

只返回 JSON 格式，不要其他内容。`, entityTypeCN)

	userPrompt := fmt.Sprintf(`输入名称: "%s"
候选列表: %v

请从候选列表中选择最匹配的项，返回 JSON:
{"matched": "匹配的候选名称", "confidence": 0.0-1.0, "reason": "匹配原因"}
如果没有匹配项，返回: {"matched": null, "confidence": 0, "reason": "原因"}`, input, candidateNames)

	resp, err := m.aiFactory.CallDefault(ctx, systemPrompt, userPrompt, nil)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 解析响应
	var result struct {
		Matched    *string `json:"matched"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}

	content := strings.TrimSpace(resp.Content)
	// 提取 JSON
	if idx := strings.Index(content, "{"); idx >= 0 {
		if endIdx := strings.LastIndex(content, "}"); endIdx > idx {
			content = content[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if result.Matched == nil || *result.Matched == "" || result.Confidence < 0.5 {
		return "", fmt.Errorf("LLM 未找到匹配: %s", result.Reason)
	}

	matchedID, ok := nameToID[*result.Matched]
	if !ok {
		return "", fmt.Errorf("LLM 返回的名称不在候选列表中: %s", *result.Matched)
	}

	m.logger.Info("LLM 名称匹配结果",
		"input", input,
		"matched", *result.Matched,
		"confidence", result.Confidence,
		"reason", result.Reason)

	return matchedID, nil
}

// levenshteinDistance 计算编辑距离
func (m *NameMatcher) levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	column := make([]int, len(r1)+1)

	for y := 1; y <= len(r1); y++ {
		column[y] = y
	}

	for x := 1; x <= len(r2); x++ {
		column[0] = x
		lastDiag := x - 1
		for y := 1; y <= len(r1); y++ {
			oldDiag := column[y]
			cost := 0
			if r1[y-1] != r2[x-1] {
				cost = 1
			}
			column[y] = min(column[y]+1, column[y-1]+1, lastDiag+cost)
			lastDiag = oldDiag
		}
	}

	return column[len(r1)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
