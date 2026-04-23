package debug

import "testing"

// TestStoreAddAndList 验证调试存储可以保存请求摘要
func TestStoreAddAndList(t *testing.T) {
	store := NewStore(3)
	store.Add(RequestSummary{
		RequestID:  "req-1",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		StatusCode: 200,
	})

	items := store.List()
	if len(items) != 1 {
		t.Fatalf("期望请求摘要数量为 1，实际得到 %d", len(items))
	}
	if items[0].RequestID != "req-1" {
		t.Fatalf("期望请求 ID 为 req-1，实际得到 %q", items[0].RequestID)
	}
}

// TestStoreCapacity 验证调试存储会保留最近 N 条请求摘要
func TestStoreCapacity(t *testing.T) {
	store := NewStore(2)
	store.Add(RequestSummary{RequestID: "req-1"})
	store.Add(RequestSummary{RequestID: "req-2"})
	store.Add(RequestSummary{RequestID: "req-3"})

	items := store.List()
	if len(items) != 2 {
		t.Fatalf("期望请求摘要数量为 2，实际得到 %d", len(items))
	}
	if items[0].RequestID != "req-2" || items[1].RequestID != "req-3" {
		t.Fatalf("期望保留最近两条请求摘要，实际得到 %#v", items)
	}
}
