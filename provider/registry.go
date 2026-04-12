package provider

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Becks723/mind-gateway/core/schema"
)

// Registry 管理 Provider 的注册与查询
type Registry struct {
	mu        sync.RWMutex               // mu 表示 Provider 注册表的读写锁
	providers map[string]schema.Provider // providers 表示按名称索引的 Provider 映射
}

// NewRegistry 创建新的 Provider 注册表
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]schema.Provider),
	}
}

// Register 注册一个新的 Provider
func (r *Registry) Register(provider schema.Provider) error {
	// 校验 Provider 基本信息
	if provider == nil {
		return fmt.Errorf("provider 不能为空")
	}
	if provider.Name() == "" {
		return fmt.Errorf("provider 名称不能为空")
	}

	// 写入注册表，禁止重复名称
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[provider.Name()]; exists {
		return fmt.Errorf("provider %q 已存在", provider.Name())
	}

	r.providers[provider.Name()] = provider
	return nil
}

// Get 按名称获取 Provider
func (r *Registry) Get(name string) (schema.Provider, bool) {
	// 从注册表中读取 Provider
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	return provider, ok
}

// MustGet 按名称获取 Provider，不存在时返回错误
func (r *Registry) MustGet(name string) (schema.Provider, error) {
	// 读取并校验 Provider 是否存在
	provider, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("provider %q 不存在", name)
	}

	return provider, nil
}

// List 返回已注册的 Provider 名称列表
func (r *Registry) List() []string {
	// 收集并排序 Provider 名称
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}
