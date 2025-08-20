package models

// APIResponse 标准API响应结构体
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// RegisterSuccessData 注册成功时的数据结构
type RegisterSuccessData struct {
	ServiceName          string `json:"serviceName"`
	ServiceID            string `json:"serviceId"`
	IPAddress            string `json:"ipAddress"`
	Port                 int    `json:"port"`
	RegistrationTime     int64  `json:"registrationTime,omitempty"`
	LastHeartbeatTime    int64  `json:"lastHeartbeatTime,omitempty"`
	RegistrationGMTTime  string `json:"registrationGMTTime,omitempty"`
	LastHeartbeatGMTTime string `json:"lastHeartbeatGMTTime,omitempty"`
}

// RegisterConflictData 注册冲突时的数据结构
type RegisterConflictData struct {
	ConflictServiceID string `json:"conflictServiceId"`
	Suggestion        string `json:"suggestion,omitempty"`
}

// UnregisterSuccessData 注销成功时的数据结构
type UnregisterSuccessData struct {
	ServiceName          string `json:"serviceName"`
	ServiceID            string `json:"serviceId"`
	IPAddress            string `json:"ipAddress"`
	Port                 int    `json:"port"`
	Message              string `json:"message"`
	RegistrationGMTTime  string `json:"registrationGMTTime,omitempty"`
	LastHeartbeatGMTTime string `json:"lastHeartbeatGMTTime,omitempty"`
}

// UnregisterErrorData 注销错误时的数据结构
type UnregisterErrorData struct {
	ServiceID     string `json:"serviceId,omitempty"`
	FieldMismatch string `json:"fieldMismatch,omitempty"`
	Suggestion    string `json:"suggestion,omitempty"`
}

// DiscoverySuccessData 服务发现成功时的数据结构
type DiscoverySuccessData struct {
	ServiceName string                  `json:"serviceName,omitempty"`
	TotalCount  int                     `json:"totalCount"`
	Instances   []DiscoveryInstanceData `json:"instances"`
}

// DiscoveryInstanceData 服务实例数据
type DiscoveryInstanceData struct {
	ServiceName          string `json:"serviceName"`
	ServiceID            string `json:"serviceId"`
	IPAddress            string `json:"ipAddress"`
	Port                 int    `json:"port"`
	RegistrationTime     int64  `json:"registrationTime"`
	LastHeartbeatTime    int64  `json:"lastHeartbeatTime"`
	RegistrationGMTTime  string `json:"registrationGMTTime"`
	LastHeartbeatGMTTime string `json:"lastHeartbeatGMTTime"`
}

// DiscoveryErrorData 服务发现错误时的数据结构
type DiscoveryErrorData struct {
	ServiceName string `json:"serviceName,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// 成功响应
func SuccessResponse(code int, msg string, data interface{}) APIResponse {
	return APIResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}

// 错误响应
func ErrorResponse(code int, msg string, data interface{}) APIResponse {
	return APIResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
