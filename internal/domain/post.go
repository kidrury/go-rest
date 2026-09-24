package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("post not found")
	ErrIDRequired      = errors.New("id required")
	ErrTitleRequired   = errors.New("title required")
	ErrContentRequired = errors.New("content equired")
)

type Post struct {
	ID         string
	Title      string
	Content    string
	AuthoredBy string
	CreatedAt  time.Time
}

func (p *Post) Validate() error {
	if p.Title == "" {
		return ErrTitleRequired
	}

	if p.Content == "" {
		return ErrContentRequired
	}

	return nil
}

type PostService interface {
	CreatePost(title, content, authorId string) (*Post, error)
	GetPost(id string) (*Post, error)
	ListPosts() ([]*Post, error)
	DeletePost(id string) error
}

type PostRepository interface {
	Create(post *Post) error
	GetById(id string) (*Post, error)
	List() ([]*Post, error)
	Delete(id string) error
}
