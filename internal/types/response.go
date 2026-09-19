package types

// ApiResponse is the envelope every HTTP handler response is wrapped in by
// [middleware.ApiResponseMiddleware].
type ApiResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data"`
}
