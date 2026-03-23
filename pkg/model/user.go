package model

type User struct {
	ExpireTime    int64    `json:"expireTime"`
	LoginTime     int64    `json:"loginTime"`
	Os            string   `json:"os"`
	Permissions   []string `json:"permissions"`
	Browser       string   `json:"browser"`
	SysUser       SysUser  `json:"sysUser"`
	IpAddr        string   `json:"ipaddr"`
	LoginLocation string   `json:"loginLocation"`
	UserId        int64    `json:"userId"`
	Token         string   `json:"token"`
	UserName      string   `json:"username"`
}

type Role struct {
	Flag     bool   `json:"flag"`
	PlatId   int64  `json:"platId"`
	RoleId   int64  `json:"roleId"`
	RoleName string `json:"roleName"`
	Admin    bool   `json:"admin"`
	// GroupList []string `json:"groupList"`
	RoleType string `json:"roleType"`
	Status   string `json:"status"`
}

type SysUser struct {
	NickName    string `json:"nickName"`
	Roles       []Role `json:"roles"`
	Admin       bool   `json:"admin"`
	LoginDate   int64  `json:"loginDate"`
	DelFlag     string `json:"delFlag"`
	UserName    string `json:"userName"`
	UserId      int64  `json:"userId"`
	CreateBy    int64  `json:"createBy"`
	Password    string `json:"password"`
	PhoneNumber string `json:"phoneNumber"`
	PlatId      int64  `json:"platId"`
	WorkNumber  string `json:"workNumber"`
	CreateTime  int64  `json:"createTime"`
	LoginIp     string `json:"loginIp"`
	Status      string `json:"status"`
}
