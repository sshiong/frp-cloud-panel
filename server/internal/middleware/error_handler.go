package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// ErrorCode 错误码
type ErrorCode int

const (
	// 通用错误码
	ErrCodeSuccess         ErrorCode = 0
	ErrCodeBadRequest      ErrorCode = 400
	ErrCodeUnauthorized    ErrorCode = 401
	ErrCodeForbidden       ErrorCode = 403
	ErrCodeNotFound        ErrorCode = 404
	ErrCodeInternalError   ErrorCode = 500
	ErrCodeTooManyRequests ErrorCode = 429

	// 业务错误码
	ErrCodeUserNotFound      ErrorCode = 1001
	ErrCodeUserDisabled      ErrorCode = 1002
	ErrCodeInvalidPassword   ErrorCode = 1003
	ErrCodeUserAlreadyExists ErrorCode = 1004

	ErrCodeClientNotFound ErrorCode = 2001
	ErrCodeClientDisabled ErrorCode = 2002

	ErrCodeMappingNotFound ErrorCode = 3001
	ErrCodeMappingConflict ErrorCode = 3002
	ErrCodePortNotAvailable ErrorCode = 3003

	ErrCodeDomainNotFound ErrorCode = 4001
	ErrCodeDomainConflict ErrorCode = 4002

	ErrCodeTokenInvalid ErrorCode = 5001
	ErrCodeTokenExpired ErrorCode = 5002

	ErrCodeConfigSyncFailed ErrorCode = 6001
	ErrCodeConfigApplyFailed ErrorCode = 6002

	ErrCodeBackupFailed  ErrorCode = 7001
	ErrCodeRestoreFailed ErrorCode = 7002

	ErrCodeCertRequestFailed ErrorCode = 8001
	ErrCodeCertRenewFailed   ErrorCode = 8002
)

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code      ErrorCode   `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 记录错误日志
				log.Printf("Panic recovered: %v\n%s", r, debug.Stack())

				// 返回内部错误
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Code:    ErrCodeInternalError,
					Message: "Internal server error",
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}

// ErrorHandlerFunc 错误处理函数
type ErrorHandlerFunc func(c *gin.Context) error

// WrapHandler 包装处理器，自动处理错误
func WrapHandler(handler ErrorHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := handler(c); err != nil {
			handleError(c, err)
		}
	}
}

// handleError 处理错误
func handleError(c *gin.Context, err error) {
	// 根据错误类型返回不同的响应
	switch e := err.(type) {
	case *AppError:
		c.JSON(e.HTTPCode, ErrorResponse{
			Code:      e.Code,
			Message:   e.Message,
			Details:   e.Details,
			RequestID: c.GetString("request_id"),
		})
	case *ValidationError:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:      ErrCodeBadRequest,
			Message:   "Validation failed",
			Details:   e.Errors,
			RequestID: c.GetString("request_id"),
		})
	default:
		log.Printf("Unhandled error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:      ErrCodeInternalError,
			Message:   "Internal server error",
			RequestID: c.GetString("request_id"),
		})
	}
}

// AppError 应用错误
type AppError struct {
	Code     ErrorCode   `json:"code"`
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
	HTTPCode int         `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	return fmt.Sprintf("AppError: %d - %s", e.Code, e.Message)
}

// NewAppError 创建应用错误
func NewAppError(code ErrorCode, message string, httpCode int) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		HTTPCode: httpCode,
	}
}

// ValidationError 验证错误
type ValidationError struct {
	Errors map[string]string `json:"errors"`
}

// Error 实现 error 接口
func (e *ValidationError) Error() string {
	return "Validation failed"
}

// NewValidationError 创建验证错误
func NewValidationError(errors map[string]string) *ValidationError {
	return &ValidationError{
		Errors: errors,
	}
}

// BadRequestError 创建400错误
func BadRequestError(message string) *AppError {
	return NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// UnauthorizedError 创建401错误
func UnauthorizedError(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// ForbiddenError 创建403错误
func ForbiddenError(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden)
}

// NotFoundError 创建404错误
func NotFoundError(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message, http.StatusNotFound)
}

// InternalError 创建500错误
func InternalError(message string) *AppError {
	return NewAppError(ErrCodeInternalError, message, http.StatusInternalServerError)
}

// TooManyRequestsError 创建429错误
func TooManyRequestsError(message string) *AppError {
	return NewAppError(ErrCodeTooManyRequests, message, http.StatusTooManyRequests)
}

// UserNotFoundError 创建用户不存在错误
func UserNotFoundError() *AppError {
	return NewAppError(ErrCodeUserNotFound, "User not found", http.StatusNotFound)
}

// UserDisabledError 创建用户已禁用错误
func UserDisabledError() *AppError {
	return NewAppError(ErrCodeUserDisabled, "User is disabled", http.StatusForbidden)
}

// InvalidPasswordError 创建密码错误
func InvalidPasswordError() *AppError {
	return NewAppError(ErrCodeInvalidPassword, "Invalid password", http.StatusUnauthorized)
}

// UserAlreadyExistsError 创建用户已存在错误
func UserAlreadyExistsError() *AppError {
	return NewAppError(ErrCodeUserAlreadyExists, "User already exists", http.StatusConflict)
}

// ClientNotFoundError 创建客户端不存在错误
func ClientNotFoundError() *AppError {
	return NewAppError(ErrCodeClientNotFound, "Client not found", http.StatusNotFound)
}

// ClientDisabledError 创建客户端已禁用错误
func ClientDisabledError() *AppError {
	return NewAppError(ErrCodeClientDisabled, "Client is disabled", http.StatusForbidden)
}

// MappingNotFoundError 创建映射不存在错误
func MappingNotFoundError() *AppError {
	return NewAppError(ErrCodeMappingNotFound, "Mapping not found", http.StatusNotFound)
}

// MappingConflictError 创建映射冲突错误
func MappingConflictError() *AppError {
	return NewAppError(ErrCodeMappingConflict, "Mapping conflict", http.StatusConflict)
}

// PortNotAvailableError 创建端口不可用错误
func PortNotAvailableError() *AppError {
	return NewAppError(ErrCodePortNotAvailable, "Port not available", http.StatusConflict)
}

// DomainNotFoundError 创建域名不存在错误
func DomainNotFoundError() *AppError {
	return NewAppError(ErrCodeDomainNotFound, "Domain not found", http.StatusNotFound)
}

// DomainConflictError 创建域名冲突错误
func DomainConflictError() *AppError {
	return NewAppError(ErrCodeDomainConflict, "Domain conflict", http.StatusConflict)
}

// TokenInvalidError 创建Token无效错误
func TokenInvalidError() *AppError {
	return NewAppError(ErrCodeTokenInvalid, "Invalid token", http.StatusUnauthorized)
}

// TokenExpiredError 创建Token过期错误
func TokenExpiredError() *AppError {
	return NewAppError(ErrCodeTokenExpired, "Token expired", http.StatusUnauthorized)
}

// ConfigSyncFailedError 创建配置同步失败错误
func ConfigSyncFailedError() *AppError {
	return NewAppError(ErrCodeConfigSyncFailed, "Config sync failed", http.StatusInternalServerError)
}

// ConfigApplyFailedError 创建配置应用失败错误
func ConfigApplyFailedError() *AppError {
	return NewAppError(ErrCodeConfigApplyFailed, "Config apply failed", http.StatusInternalServerError)
}

// BackupFailedError 创建备份失败错误
func BackupFailedError() *AppError {
	return NewAppError(ErrCodeBackupFailed, "Backup failed", http.StatusInternalServerError)
}

// RestoreFailedError 创建恢复失败错误
func RestoreFailedError() *AppError {
	return NewAppError(ErrCodeRestoreFailed, "Restore failed", http.StatusInternalServerError)
}

// CertRequestFailedError 创建证书申请失败错误
func CertRequestFailedError() *AppError {
	return NewAppError(ErrCodeCertRequestFailed, "Certificate request failed", http.StatusInternalServerError)
}

// CertRenewFailedError 创建证书续期失败错误
func CertRenewFailedError() *AppError {
	return NewAppError(ErrCodeCertRenewFailed, "Certificate renewal failed", http.StatusInternalServerError)
}

// RequestIDMiddleware 请求 ID 中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成请求 ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
