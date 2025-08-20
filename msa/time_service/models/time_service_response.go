package models

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func SuccessResponse(msg string, data interface{}) Response {
	return Response{
		Code: 200,
		Msg:  msg,
		Data: data,
	}
}

func ErrorResponse(code int, msg string, data interface{}) Response {
	return Response{
		Code: code,
		Msg:  msg,
		Data: data,
	}
}
