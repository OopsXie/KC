package models

type LogEntry struct {
	ServiceName string `json:"serviceName"`
	ServiceId   string `json:"serviceId"`
	Datetime    string `json:"datetime"`
	Level       string `json:"level"`
	Message     string `json:"message"`
}

type LogResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data LogEntry `json:"data"`
}
