package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/auth"
)

type authRepo struct {
	db *sqlx.DB
}

func NewAuthRepository(
	db *sqlx.DB,
) auth.AuthRepository {
	return &authRepo{
		db: db,
	}
}

func (r *authRepo) GetByEmail(ctx context.Context, email string) (auth.User, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from users where email = $1`
	var user auth.User
	if err := sqlx.GetContext(ctx, db, &user, query, email); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (r *authRepo) GetById(ctx context.Context, id uuid.UUID) (auth.User, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from users where id = $1`
	var user auth.User
	if err := sqlx.GetContext(ctx, db, &user, query, id); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (r *authRepo) Create(ctx context.Context, user auth.User) (auth.User, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		users(username, fullname, email, password_hash, is_verified, role, deleted_at)
		values(:username, :fullname, :email, :password_hash, :is_verified, :role, :deleted_at)
		returning id, username, fullname, email, is_verified, role, created_at
	`

	rows, err := sqlx.NamedQueryContext(ctx, db, query, user)
	if err != nil {
		return auth.User{}, err
	}
	defer rows.Close()

	var created auth.User
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return auth.User{}, err
		}
	}
	return created, nil
}

func (r *authRepo) DeleteById(ctx context.Context, id uuid.UUID) error {
	db := getDBFromCtx(ctx, r.db)
	query := `delete from users where id = $1`
	_, err := db.ExecContext(ctx, query, id)
	return err
}

func (r *authRepo) DeleteByEmail(ctx context.Context, email string) error {
	db := getDBFromCtx(ctx, r.db)
	query := `delete from users where email = $1`
	_, err := db.ExecContext(ctx, query, email)
	return err
}

func (r *authRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passHash string) error {
	db := getDBFromCtx(ctx, r.db)
	query := `update users set password_hash = $1 where id = $2`
	_, err := db.ExecContext(ctx, query, passHash, id)
	return err
}

func (r *authRepo) SetVerified(ctx context.Context, userId uuid.UUID) error {
	db := getDBFromCtx(ctx, r.db)
	query := `update users set is_verified = true where id = $1`
	_, err := db.ExecContext(ctx, query, userId)
	return err
}

func (r *authRepo) Upgrade(ctx context.Context, id string, user auth.User) (auth.User, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
	update users 
	set
		username = COALESCE($1, username),
		fullname = COALESCE($2, fullname),
		email = COALESCE($3, email),
		password_hash = COALESCE($4, password_hash),
		role = COALESCE($5, role),
		is_verified = COALESCE($6, is_verified),
		deleted_at = COALESCE($7, deleted_at),
		updated_at = NOW()
	where id = $8
	returning id, username, fullname, email, is_verified, role, created_at
	`

	var updated auth.User
	err := db.QueryRowxContext(ctx, query,
		user.Username,
		user.Fullname,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.IsVerified,
		user.DeletedAt,
		id,
	).StructScan(&updated)

	if err != nil {
		return auth.User{}, err
	}

	return updated, nil
}

// func (r *authRepo) CreateDemo(user auth.AnonUser) (auth.AnonUser, error) {
// 	query := `insert into
// 		anon_users(username, fullname)
// 		values(:username, :fullname)
// 		returning id, username, fullname, created_at
// 	`

// 	rows, err := r.dbConn.NamedQuery(query, user)
// 	if err != nil {
// 		return auth.AnonUser{}, err
// 	}
// 	defer rows.Close()

// 	var created auth.AnonUser
// 	if rows.Next() {
// 		if err := rows.StructScan(&created); err != nil {
// 			return auth.AnonUser{}, err
// 		}
// 	}
// 	return created, nil
// }

type verifierRepo struct {
	db *sqlx.DB
}

func NewVerifierRepo(
	db *sqlx.DB,
) auth.VerifierRepository {
	return &verifierRepo{
		db: db,
	}
}

func (r *verifierRepo) Create(ctx context.Context, verifier auth.Verifier) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		verifier(user_id, token_hash)
		values(:user_id, :token_hash)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, verifier)
	return err
}

func (r *verifierRepo) GetByHash(ctx context.Context, tokenHash string) (auth.Verifier, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from verifier where token_hash = $1 and expire_at > now()`
	var verifier auth.Verifier
	if err := sqlx.GetContext(ctx, db, &verifier, query, tokenHash); err != nil {
		return auth.Verifier{}, err
	}
	return verifier, nil
}

func (r *verifierRepo) GetById(ctx context.Context, userId uuid.UUID) (auth.Verifier, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from verifier where user_id = $1`
	var verifier auth.Verifier
	if err := sqlx.GetContext(ctx, db, &verifier, query, userId); err != nil {
		return auth.Verifier{}, err
	}
	return verifier, nil
}

func (r *verifierRepo) Delete(ctx context.Context, id int64) error {
	db := getDBFromCtx(ctx, r.db)
	query := `delete from verifier where id = $1`
	_, err := db.ExecContext(ctx, query, id)
	return err
}

type reseterRepo struct {
	db *sqlx.DB
}

func NewReseterRepo(
	db *sqlx.DB,
) auth.ReseterRepository {
	return &reseterRepo{
		db: db,
	}
}

func (r *reseterRepo) GetById(ctx context.Context, id uuid.UUID) (auth.Reseter, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from reseter where user_id = $1 and expire_at > now()`
	var reseter auth.Reseter
	if err := sqlx.GetContext(ctx, db, &reseter, query, id); err != nil {
		return auth.Reseter{}, err
	}
	return reseter, nil
}

// no use case
func (r *reseterRepo) Update(ctx context.Context, reseter auth.Reseter) error {
	db := getDBFromCtx(ctx, r.db)
	_ = db
	return nil
}

func (r *reseterRepo) Create(ctx context.Context, reseter auth.Reseter) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		reseter(user_id, token_hash)
		values(:user_id, :token_hash)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, reseter)
	return err
}

func (r *reseterRepo) GetByToken(ctx context.Context, token string) (auth.Reseter, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select * from reseter where token_hash = $1 and expire_at > now()`
	var reseter auth.Reseter
	if err := sqlx.GetContext(ctx, db, &reseter, query, token); err != nil {
		return auth.Reseter{}, err
	}
	return reseter, nil
}

func (r *reseterRepo) DeleteById(ctx context.Context, id int64) error {
	db := getDBFromCtx(ctx, r.db)
	query := `delete from reseter where id = $1`
	_, err := db.ExecContext(ctx, query, id)
	return err
}
