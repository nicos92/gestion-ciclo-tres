package main

import (
	"errors"
	"strings"
)

var (
	ErrEmptyText = errors.New("el texto no puede estar vacío")
	ErrNotFound  = errors.New("tarea no encontrada")
)

type todoService struct {
	repo todoRepository
}

func newTodoService(repo todoRepository) *todoService {
	return &todoService{repo: repo}
}

func (s *todoService) Create(text string) (todo, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return todo{}, ErrEmptyText
	}
	return s.repo.Add(text), nil
}

func (s *todoService) Delete(id int) error {
	if !s.repo.Remove(id) {
		return ErrNotFound
	}
	return nil
}

func (s *todoService) Toggle(id int) (todo, error) {
	item, ok := s.repo.Toggle(id)
	if !ok {
		return todo{}, ErrNotFound
	}
	return item, nil
}

func (s *todoService) List() []todo {
	return s.repo.All()
}