package service

import (
	"fmt"
	"sync"
	"testing"
)

func TestMemSessionStorage_CRUD(t *testing.T) {
	t.Parallel()

	ms := NewMemStorage()

	if v, ok := ms.GetSession("s1"); ok || v != "" {
		t.Fatalf("expected missing session, got ok=%v v=%q", ok, v)
	}

	ms.AddSession("s1", "user1")
	if v, ok := ms.GetSession("s1"); !ok || v != "user1" {
		t.Fatalf("expected ok=true v=user1, got ok=%v v=%q", ok, v)
	}

	ms.AddSession("s1", "user2")
	if v, ok := ms.GetSession("s1"); !ok || v != "user2" {
		t.Fatalf("expected overwrite to user2, got ok=%v v=%q", ok, v)
	}

	ms.DeleteSession("s1")
	if v, ok := ms.GetSession("s1"); ok || v != "" {
		t.Fatalf("expected deleted session, got ok=%v v=%q", ok, v)
	}

	ms.DeleteSession("unknown")
}

func TestMemSessionStorage_ConcurrentAccess(t *testing.T) {
	ms := NewMemStorage()

	const (
		goroutines = 50
		iterations = 200
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				id := fmt.Sprintf("s-%d-%d", g, i)
				login := fmt.Sprintf("u-%d-%d", g, i)

				ms.AddSession(id, login)

				// read after write should see value (в рамках одного goroutine)
				if v, ok := ms.GetSession(id); !ok || v != login {
					t.Fatalf("expected ok=true v=%q, got ok=%v v=%q", login, ok, v)
				}

				ms.DeleteSession(id)

				if v, ok := ms.GetSession(id); ok || v != "" {
					t.Fatalf("expected deleted, got ok=%v v=%q", ok, v)
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestMemSessionStorage_ConcurrentSameKey(t *testing.T) {
	ms := NewMemStorage()

	const (
		writers = 20
		readers = 20
		iters   = 500
	)

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for w := 0; w < writers; w++ {
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				ms.AddSession("shared", fmt.Sprintf("writer-%d-%d", w, i))
			}
		}(w)
	}

	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_, _ = ms.GetSession("shared")
			}
		}()
	}

	wg.Wait()

	if _, ok := ms.GetSession("shared"); !ok {
		t.Fatalf("expected shared session to exist after writes")
	}
}
