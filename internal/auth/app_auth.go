package auth

import "time"

const appRoleName = "app:employee"

type AppLoginInput struct {
	LoginID  string `json:"login_id"`
	Password string `json:"password"`
}

type AppEmployeeInfo struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	EmployeeNo     *string `json:"employee_no,omitempty"`
	Phone          *string `json:"phone,omitempty"`
	Email          *string `json:"email,omitempty"`
	OrgNodeID      string  `json:"org_node_id"`
	OrgNodeName    string  `json:"org_node_name"`
	MustResetPwd   bool    `json:"must_reset_pwd"`
	SchedulingRole string  `json:"scheduling_role,omitempty"`
}

type AppCurrentNodeInfo struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name"`
	NodePath string `json:"node_path"`
}

type AppLoginResponse struct {
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	ExpiresIn    int                `json:"expires_in"`
	Employee     AppEmployeeInfo    `json:"employee"`
	CurrentNode  AppCurrentNodeInfo `json:"current_node"`
	MustResetPwd bool               `json:"must_reset_pwd"`
}

type AppMeResponse struct {
	Employee    AppEmployeeInfo    `json:"employee"`
	CurrentNode AppCurrentNodeInfo `json:"current_node"`
}

type appEmployeeRecord struct {
	ID              string
	OrgNodeID       string
	OrgNodeName     string
	OrgNodePath     string
	Name            string
	EmployeeNo      *string
	Phone           *string
	Email           *string
	Status          string
	SchedulingRole  string
	AppPasswordHash *string
	AppMustResetPwd bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (r appEmployeeRecord) toInfo() AppEmployeeInfo {
	return AppEmployeeInfo{
		ID:             r.ID,
		Name:           r.Name,
		EmployeeNo:     r.EmployeeNo,
		Phone:          r.Phone,
		Email:          r.Email,
		OrgNodeID:      r.OrgNodeID,
		OrgNodeName:    r.OrgNodeName,
		MustResetPwd:   r.AppMustResetPwd,
		SchedulingRole: r.SchedulingRole,
	}
}

func (r appEmployeeRecord) currentNode() AppCurrentNodeInfo {
	return AppCurrentNodeInfo{
		NodeID:   r.OrgNodeID,
		NodeName: r.OrgNodeName,
		NodePath: r.OrgNodePath,
	}
}
