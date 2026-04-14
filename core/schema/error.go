package schema

// 通用错误类型常量
const (
	ErrorTypeAuthentication   = "authentication_error"  // ErrorTypeAuthentication 表示认证失败错误
	ErrorTypePermission       = "permission_error"      // ErrorTypePermission 表示权限不足错误
	ErrorTypeNotFound         = "not_found_error"       // ErrorTypeNotFound 表示资源未找到错误
	ErrorTypeMethodNotAllowed = "method_not_allowed"    // ErrorTypeMethodNotAllowed 表示方法不允许错误
	ErrorTypeInvalidRequest   = "invalid_request_error" // ErrorTypeInvalidRequest 表示非法请求错误
	ErrorTypeRateLimit        = "rate_limit_error"      // ErrorTypeRateLimit 表示限流或配额错误
	ErrorTypeInternal         = "internal_error"        // ErrorTypeInternal 表示内部错误
)

// 通用错误码常量
const (
	ErrorCodeVirtualKeyRequired       = "virtual_key_required"        // ErrorCodeVirtualKeyRequired 表示缺少虚拟密钥
	ErrorCodeVirtualKeyInvalid        = "virtual_key_invalid"         // ErrorCodeVirtualKeyInvalid 表示虚拟密钥无效
	ErrorCodeProviderNotAllowed       = "provider_not_allowed"        // ErrorCodeProviderNotAllowed 表示不允许访问当前 Provider
	ErrorCodeModelNotAllowed          = "model_not_allowed"           // ErrorCodeModelNotAllowed 表示不允许访问当前模型
	ErrorCodeRequestQuotaExceeded     = "request_quota_exceeded"      // ErrorCodeRequestQuotaExceeded 表示请求次数额度超限
	ErrorCodeInputTokenQuotaExceeded  = "input_token_quota_exceeded"  // ErrorCodeInputTokenQuotaExceeded 表示输入 Token 额度超限
	ErrorCodeOutputTokenQuotaExceeded = "output_token_quota_exceeded" // ErrorCodeOutputTokenQuotaExceeded 表示输出 Token 额度超限
	ErrorCodeGovernanceInternalError  = "governance_internal_error"   // ErrorCodeGovernanceInternalError 表示治理插件内部错误
)
