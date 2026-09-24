package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kidrury/rest-pro/internal/domain"
)

type postService struct {
	repo domain.PostRepository
}

func NewPostService(repo domain.PostRepository) domain.PostService {
	return &postService{
		repo: repo,
	}
}

func (s *postService) CreatePost(title, content, authorId string) (*domain.Post, error) {
	post := &domain.Post{
		ID:         uuid.New().String(),
		Title:      title,
		Content:    content,
		AuthoredBy: authorId,
		CreatedAt:  time.Now().UTC(),
	}

	if err := post.Validate(); err != nil {
		return nil, err
	}

	err := s.repo.Create(post)

	if err != nil {
		return nil, fmt.Errorf("repository.Create: %w", err)
	}

	return post, nil
}

func (s *postService) GetPost(id string) (*domain.Post, error) {
	if id == "" {
		return nil, domain.ErrIDRequired
	}

	post, err := s.repo.GetById(id)

	if err != nil {
		return nil, fmt.Errorf("repository.GetById: %w", err)
	}

	return post, nil
}

func (s *postService) ListPosts() ([]*domain.Post, error) {
	posts, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("repository.List: %w", err)
	}

	return posts, nil
}

func (s *postService) DeletePost(id string) error {
	if id == "" {
		return domain.ErrIDRequired
	}

	err := s.repo.Delete(id)

	if err != nil {
		return fmt.Errorf("repository.Delete: %w", err)
	}

	return nil
}
