package identity

import "context"

type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*Usuario, error)
	Create(ctx context.Context, u *Usuario) (int64, error)
	ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
	GetByID(ctx context.Context, id int64) (*Usuario, error)
	ListAll(ctx context.Context) ([]Usuario, error)
	Update(ctx context.Context, u *Usuario) error
	UpdatePassword(ctx context.Context, id int64, password string) error
	CountAll(ctx context.Context) (int, error)
	ExistsByUsernameOrEmailExcluding(ctx context.Context, username, email string, excludeID int64) (bool, error)
}
