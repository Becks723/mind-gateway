package tool

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Becks723/mind-gateway/core/schema"
)

// Handler 定义工具执行函数签名
type Handler func(ctx context.Context, arguments string) (string, error)

// entry 定义单个工具注册项
type entry struct {
	definition schema.ToolDefinition // definition 表示工具定义
	handler    Handler               // handler 表示工具处理函数
}

// Registry 定义工具注册表
type Registry struct {
	tools map[string]entry // tools 表示按名称索引的工具集合
	mu    sync.RWMutex     // mu 表示工具注册表读写锁
}

// NewRegistry 创建新的工具注册表
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]entry),
	}
}

// Register 注册单个工具
func (r *Registry) Register(definition schema.ToolDefinition, handler Handler) error {
	// 校验输入参数
	if r == nil {
		return fmt.Errorf("工具注册表不能为空")
	}
	if definition.Name == "" {
		return fmt.Errorf("工具名称不能为空")
	}
	if handler == nil {
		return fmt.Errorf("工具 %q 的处理函数不能为空", definition.Name)
	}

	// 写入工具注册表
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[definition.Name]; exists {
		return fmt.Errorf("工具 %q 已注册", definition.Name)
	}
	r.tools[definition.Name] = entry{
		definition: cloneToolDefinition(definition),
		handler:    handler,
	}
	return nil
}

// Definitions 返回全部已注册工具定义
func (r *Registry) Definitions() []schema.ToolDefinition {
	// 按工具名称返回稳定顺序
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]schema.ToolDefinition, 0, len(names))
	for _, name := range names {
		result = append(result, cloneToolDefinition(r.tools[name].definition))
	}
	return result
}

// AllowedDefinitions 返回允许的工具定义
func (r *Registry) AllowedDefinitions(allowedNames []string) []schema.ToolDefinition {
	// 允许列表为空时返回全部工具定义
	if len(allowedNames) == 0 {
		return r.Definitions()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]schema.ToolDefinition, 0, len(allowedNames))
	for _, name := range allowedNames {
		entryValue, ok := r.tools[name]
		if !ok {
			continue
		}
		result = append(result, cloneToolDefinition(entryValue.definition))
	}
	return result
}

// Execute 执行指定工具
func (r *Registry) Execute(ctx context.Context, name string, arguments string) (string, error) {
	// 读取工具执行器
	if r == nil {
		return "", fmt.Errorf("工具注册表不能为空")
	}

	r.mu.RLock()
	entryValue, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("工具 %q 未注册", name)
	}

	return entryValue.handler(ctx, arguments)
}

// cloneToolDefinition 复制工具定义
func cloneToolDefinition(definition schema.ToolDefinition) schema.ToolDefinition {
	return schema.ToolDefinition{
		Name:        definition.Name,
		Description: definition.Description,
		InputSchema: cloneMap(definition.InputSchema),
	}
}

// cloneMap 复制 map 结构
func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}

	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
