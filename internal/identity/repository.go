package identity

import "context"

type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*Usuario, error)
	Create(ctx context.Context, u *Usuario) (int64, error)
	ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
}
