package websocket

import (
"sync"
"testing"
)

func TestHub_RegisterAndBroadcast(t *testing.T) {
hub := NewHub()

c1 := &Conn{ID: "c1", GroupID: "g1", Send: make(chan []byte, 10)}
c2 := &Conn{ID: "c2", GroupID: "g1", Send: make(chan []byte, 10)}
c3 := &Conn{ID: "c3", GroupID: "g2", Send: make(chan []byte, 10)}

hub.Register(c1)
hub.Register(c2)
hub.Register(c3)

if hub.ConnCount() != 3 {
t.Fatalf("expected 3 conns, got %d", hub.ConnCount())
}
if hub.GroupCount("g1") != 2 {
t.Fatalf("expected 2 in g1, got %d", hub.GroupCount("g1"))
}

// 向 g1 广播
msg := map[string]string{"type": "test"}
if err := hub.BroadcastToGroup("g1", msg); err != nil {
t.Fatal(err)
}

// c1 和 c2 应收到消息
if len(c1.Send) != 1 {
t.Errorf("c1 expected 1 msg, got %d", len(c1.Send))
}
if len(c2.Send) != 1 {
t.Errorf("c2 expected 1 msg, got %d", len(c2.Send))
}
// c3 不应收到
if len(c3.Send) != 0 {
t.Errorf("c3 expected 0 msg, got %d", len(c3.Send))
}
}

func TestHub_BroadcastAll(t *testing.T) {
hub := NewHub()

c1 := &Conn{ID: "c1", GroupID: "g1", Send: make(chan []byte, 10)}
c2 := &Conn{ID: "c2", GroupID: "g2", Send: make(chan []byte, 10)}

hub.Register(c1)
hub.Register(c2)

if err := hub.BroadcastAll("hello"); err != nil {
t.Fatal(err)
}
if len(c1.Send) != 1 || len(c2.Send) != 1 {
t.Error("expected both conns to receive broadcast")
}
}

func TestHub_Unregister(t *testing.T) {
hub := NewHub()

c1 := &Conn{ID: "c1", GroupID: "g1", Send: make(chan []byte, 10)}
hub.Register(c1)

hub.Unregister(c1)
if hub.ConnCount() != 0 {
t.Errorf("expected 0 conns after unregister, got %d", hub.ConnCount())
}
if hub.GroupCount("g1") != 0 {
t.Errorf("expected 0 in g1 after unregister, got %d", hub.GroupCount("g1"))
}
}

func TestHub_ConcurrentSafe(t *testing.T) {
hub := NewHub()
var wg sync.WaitGroup

for i := 0; i < 50; i++ {
wg.Add(1)
go func(id int) {
defer wg.Done()
c := &Conn{ID: "c", GroupID: "g1", Send: make(chan []byte, 100)}
hub.Register(c)
_ = hub.BroadcastToGroup("g1", "msg")
hub.Unregister(c)
}(i)
}
wg.Wait()
// 没有 panic 就是通过
}
