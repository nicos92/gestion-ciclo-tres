package main

import "sync"

type todoRepository interface {
	Add(text string) todo
	Remove(id int) bool
	Toggle(id int) (todo, bool)
	All() []todo
}

type inMemoryTodoRepository struct {
	mu    sync.Mutex
	items []todo
	next  int
}

func newInMemoryTodoRepository() *inMemoryTodoRepository {
	return &inMemoryTodoRepository{}
}

func (r *inMemoryTodoRepository) Add(text string) todo {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	item := todo{ID: r.next, Text: text}
	r.items = append(r.items, item)
	return item
}

func (r *inMemoryTodoRepository) Remove(id int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, it := range r.items {
		if it.ID == id {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return true
		}
	}
	return false
}

func (r *inMemoryTodoRepository) Toggle(id int) (todo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].Done = !r.items[i].Done
			return r.items[i], true
		}
	}
	return todo{}, false
}

func (r *inMemoryTodoRepository) All() []todo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]todo(nil), r.items...)
}
