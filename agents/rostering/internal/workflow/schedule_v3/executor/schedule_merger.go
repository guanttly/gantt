package executor

import (
	"encoding/json"
	"fmt"
	"strings"

	d_model "jusha/agent/rostering/domain/model"
)

// mergeScheduleOutput 合并排班输出到草案
func (e *ProgressiveTaskExecutor) mergeScheduleOutput(
	draft *d_model.ShiftScheduleDraft,
	output *d_model.ScheduleOutput,
	fixedAssignments map[string][]string,
) error {
	if output == nil || output.Schedule == nil {
		return nil
	}

	if draft.Schedule == nil {
		draft.Schedule = make(map[string][]string)
	}

	for date, staffIDs := range output.Schedule {
		// 转换shortID为UUID（统一通过 taskContext.ResolveStaffID）
		resolvedIDs := make([]string, 0, len(staffIDs))
		for _, id := range staffIDs {
			resolvedIDs = append(resolvedIDs, e.taskContext.ResolveStaffID(id))
		}

		switch output.Mode {
		case d_model.ScheduleOutputModeReplace:
			// 替换模式：先获取固定排班人员
			fixedStaff := make(map[string]bool)
			if fixed, ok := fixedAssignments[date]; ok {
				for _, fid := range fixed {
					fixedStaff[fid] = true
				}
			}

			// 合并固定排班 + 新输出（去重）
			merged := make([]string, 0)
			staffMap := make(map[string]bool)

			// 先加入固定排班
			for fid := range fixedStaff {
				if !staffMap[fid] {
					merged = append(merged, fid)
					staffMap[fid] = true
				}
			}

			// 再加入LLM输出（跳过已存在的）
			for _, id := range resolvedIDs {
				if !staffMap[id] {
					merged = append(merged, id)
					staffMap[id] = true
				}
			}

			draft.Schedule[date] = merged

		case d_model.ScheduleOutputModeAdd:
			fallthrough
		default:
			// 追加模式（默认）：将新人员追加到已有排班
			existing := draft.Schedule[date]
			staffMap := make(map[string]bool)
			for _, id := range existing {
				staffMap[id] = true
			}
			for _, id := range resolvedIDs {
				if !staffMap[id] {
					draft.Schedule[date] = append(draft.Schedule[date], id)
					staffMap[id] = true
				}
			}
		}
	}

	return nil
}

// parseScheduleOutput 解析LLM响应为统一的ScheduleOutput结构
// 兼容旧格式（无mode字段时默认为add）
func (e *ProgressiveTaskExecutor) parseScheduleOutput(
	response string,
	taskID string,
) (*d_model.ScheduleOutput, error) {
	// 提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in AI response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	// 尝试解析为统一格式
	var output d_model.ScheduleOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		// 尝试兼容旧格式（只有schedule和reasoning）
		var oldFormat struct {
			Schedule  map[string][]string `json:"schedule"`
			Reasoning string              `json:"reasoning,omitempty"`
		}
		if err2 := json.Unmarshal([]byte(jsonStr), &oldFormat); err2 != nil {
			return nil, fmt.Errorf("failed to parse AI response JSON: %w", err)
		}
		// 转换为新格式，默认add模式
		output = d_model.ScheduleOutput{
			Mode:      d_model.ScheduleOutputModeAdd,
			Schedule:  oldFormat.Schedule,
			Reasoning: oldFormat.Reasoning,
		}
	}

	// 如果mode为空，默认为add
	if output.Mode == "" {
		output.Mode = d_model.ScheduleOutputModeAdd
	}

	return &output, nil
}
