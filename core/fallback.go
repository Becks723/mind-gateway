package core

// resolveProviderChain 返回主 Provider 和 fallback Provider 的执行链
func (g *Gateway) resolveProviderChain(providerName string) []string {
	// 组装主 Provider 和 fallback 列表
	result := []string{providerName}
	for _, fallbackProvider := range g.providerFallbacks[providerName] {
		if fallbackProvider == "" || fallbackProvider == providerName {
			continue
		}
		result = append(result, fallbackProvider)
	}

	return result
}

// shouldTryFallback 判断当前错误是否允许继续尝试 fallback
func (g *Gateway) shouldTryFallback(err error) bool {
	if err == nil {
		return false
	}
	if IsNonRetryable(err) {
		return false
	}

	return true
}
