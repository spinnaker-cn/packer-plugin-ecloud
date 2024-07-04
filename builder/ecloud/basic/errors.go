package basic

import "fmt"

type EcloudSDKError struct {
	Code      string
	Message   string
	RequestId string
}

func (e *EcloudSDKError) Error() string {
	if e.RequestId == "" {
		return fmt.Sprintf("[TencentCloudSDKError] Code=%s, Message=%s", e.Code, e.Message)
	}
	return fmt.Sprintf("[TencentCloudSDKError] Code=%s, Message=%s, RequestId=%s", e.Code, e.Message, e.RequestId)
}

func NewTencentCloudSDKError(code, message, requestId string) error {
	return &EcloudSDKError{
		Code:      code,
		Message:   message,
		RequestId: requestId,
	}
}

func (e *EcloudSDKError) GetCode() string {
	return e.Code
}

func (e *EcloudSDKError) GetMessage() string {
	return e.Message
}

func (e *EcloudSDKError) GetRequestId() string {
	return e.RequestId
}
