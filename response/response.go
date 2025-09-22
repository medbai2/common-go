package response

import (
	"net/http"
	"time"

	"go-common/errors"

	"github.com/gin-gonic/gin"
)

// Response represents a standardized API response
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorInfo represents error information in the response
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Meta represents metadata in the response
type Meta struct {
	RequestID string `json:"requestId,omitempty"`
	Version   string `json:"version,omitempty"`
	Page      *Page  `json:"page,omitempty"`
}

// Page represents pagination information
type Page struct {
	Current  int `json:"current"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
	Pages    int `json:"pages"`
}

// Success creates a successful response
func Success(c *gin.Context, data interface{}, version string) {
	response := Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Add metadata if available
	response.Meta = &Meta{
		RequestID: getRequestID(c),
		Version:   version,
	}

	c.JSON(http.StatusOK, response)
}

// SuccessWithMeta creates a successful response with custom metadata
func SuccessWithMeta(c *gin.Context, data interface{}, meta *Meta, version string) {
	response := Response{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Ensure request ID is set
	if response.Meta != nil && response.Meta.RequestID == "" {
		response.Meta.RequestID = getRequestID(c)
	} else if response.Meta == nil {
		response.Meta = &Meta{
			RequestID: getRequestID(c),
			Version:   version,
		}
	}

	c.JSON(http.StatusOK, response)
}

// Error creates an error response
func Error(c *gin.Context, err error, version string) {
	appErr := errors.GetAppError(err)
	if appErr == nil {
		// Convert generic error to AppError
		appErr = errors.NewInternalError(err)
	}

	response := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Details: appErr.Details,
		},
		Meta: &Meta{
			RequestID: getRequestID(c),
			Version:   version,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(appErr.HTTPStatus, response)
}

// ErrorWithCode creates an error response with a specific error code
func ErrorWithCode(c *gin.Context, code errors.ErrorCode, message string, version string) {
	appErr := errors.New(code, message)
	Error(c, appErr, version)
}

// ErrorWithDetails creates an error response with details
func ErrorWithDetails(c *gin.Context, code errors.ErrorCode, message, details string, version string) {
	appErr := errors.NewWithDetails(code, message, details)
	Error(c, appErr, version)
}

// ValidationError creates a validation error response
func ValidationError(c *gin.Context, message string, version string) {
	ErrorWithCode(c, errors.ErrCodeInvalidInput, message, version)
}

// NotFound creates a not found error response
func NotFound(c *gin.Context, resource string, version string) {
	ErrorWithCode(c, errors.ErrCodeNotFound, resource+" not found", version)
}

// InternalError creates an internal server error response
func InternalError(c *gin.Context, err error, version string) {
	Error(c, errors.NewInternalError(err), version)
}

// ServiceUnavailable creates a service unavailable error response
func ServiceUnavailable(c *gin.Context, service string, version string) {
	ErrorWithCode(c, errors.ErrCodeServiceUnavailable, service+" service is temporarily unavailable", version)
}

// BadRequest creates a bad request error response
func BadRequest(c *gin.Context, message string, version string) {
	ErrorWithCode(c, errors.ErrCodeInvalidInput, message, version)
}

// Unauthorized creates an unauthorized error response
func Unauthorized(c *gin.Context, message string, version string) {
	ErrorWithCode(c, errors.ErrCodeUnauthorized, message, version)
}

// Forbidden creates a forbidden error response
func Forbidden(c *gin.Context, message string, version string) {
	ErrorWithCode(c, errors.ErrCodeForbidden, message, version)
}

// TooManyRequests creates a rate limit error response
func TooManyRequests(c *gin.Context, message string, version string) {
	ErrorWithCode(c, errors.ErrCodeRateLimit, message, version)
}

// getRequestID extracts request ID from context
func getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("requestId"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// Paginated creates a paginated response
func Paginated(c *gin.Context, data interface{}, current, pageSize, total int, version string) {
	pages := (total + pageSize - 1) / pageSize // Calculate total pages

	meta := &Meta{
		RequestID: getRequestID(c),
		Version:   version,
		Page: &Page{
			Current:  current,
			PageSize: pageSize,
			Total:    total,
			Pages:    pages,
		},
	}

	SuccessWithMeta(c, data, meta, version)
}

// HealthCheck creates a health check response
func HealthCheck(c *gin.Context, status string, details map[string]interface{}, version string) {
	healthData := map[string]interface{}{
		"status": status,
	}

	if details != nil {
		healthData["details"] = details
	}

	Success(c, healthData, version)
}
