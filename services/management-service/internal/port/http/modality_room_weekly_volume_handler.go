package http

import (
	"encoding/json"
	"net/http"

	"jusha/gantt/service/management/domain/model"

	"github.com/gorilla/mux"
)

// GetModalityRoomWeeklyVolumes 获取机房周检查量配置
func (h *HTTPHandler) GetModalityRoomWeeklyVolumes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modalityRoomID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	result, err := h.container.ModalityRoomWeeklyVolumeService().GetWeeklyVolumes(r.Context(), orgID, modalityRoomID)
	if err != nil {
		h.logger.Error("Failed to get weekly volumes", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, result)
}

// SaveModalityRoomWeeklyVolumes 保存机房周检查量配置
func (h *HTTPHandler) SaveModalityRoomWeeklyVolumes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modalityRoomID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	var items []*model.WeeklyVolumeItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		RespondBadRequest(w, "Invalid request body")
		return
	}

	req := &model.WeeklyVolumeSaveRequest{
		ModalityRoomID: modalityRoomID,
		Items:          items,
	}

	if err := h.container.ModalityRoomWeeklyVolumeService().SaveWeeklyVolumes(r.Context(), orgID, req); err != nil {
		h.logger.Error("Failed to save weekly volumes", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "保存成功"})
}

// DeleteModalityRoomWeeklyVolumes 删除机房周检查量配置
func (h *HTTPHandler) DeleteModalityRoomWeeklyVolumes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	modalityRoomID := vars["id"]
	orgID := r.URL.Query().Get("orgId")

	if orgID == "" {
		RespondBadRequest(w, "orgId is required")
		return
	}

	if err := h.container.ModalityRoomWeeklyVolumeService().DeleteWeeklyVolumes(r.Context(), orgID, modalityRoomID); err != nil {
		h.logger.Error("Failed to delete weekly volumes", "error", err)
		RespondInternalError(w, err.Error())
		return
	}

	RespondSuccess(w, map[string]string{"message": "删除成功"})
}
