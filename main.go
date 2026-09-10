package main

import "net/http"

func main() {
	repo := newInMemoryTodoRepository()
	service := newTodoService(repo)
	handlers := newTodoHandlers(service)

	mux := http.NewServeMux()
	handlers.register(mux)

	http.ListenAndServe(":8080", mux)
}