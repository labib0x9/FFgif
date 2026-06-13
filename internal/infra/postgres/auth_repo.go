package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/internal/domain/auth"
)

var ctx = context.Background()

type authRepo struct {
	dbConn *sqlx.DB
}

func NewAuthRepository(
	dbConn *sqlx.DB,
) auth.AuthRepository {
	return &authRepo{
		dbConn: dbConn,
	}
}

func (r *authRepo) GetByEmail(email string) (auth.User, error) {
	query := `select * from users where email = $1`
	var user auth.User
	if err := r.dbConn.Get(&user, query, email); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (r *authRepo) GetById(id uuid.UUID) (auth.User, error) {
	query := `select * from users where id = $1`
	var user auth.User
	if err := r.dbConn.Get(&user, query, id); err != nil {
		return auth.User{}, err
	}
	return user, nil
}

func (r *authRepo) Create(user auth.User) (auth.User, error) {
	query := `insert into 
		users(username, fullname, email, password_hash, is_verified, role, deleted_at)
		values(:username, :fullname, :email, :password_hash, :is_verified, :role, :deleted_at)
		returning id, username, fullname, email, is_verified, role, created_at
	`

	rows, err := r.dbConn.NamedQuery(query, user)
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

func (r *authRepo) DeleteById(id uuid.UUID) error {
	query := `delete from users where id = $1`
	_, err := r.dbConn.Exec(query, id)
	return err
}

func (r *authRepo) DeleteByEmail(email string) error {
	query := `delete from users where email = $1`
	_, err := r.dbConn.Exec(query, email)
	return err
}

func (r *authRepo) UpdatePassword(id uuid.UUID, passHash string) error {
	query := `update users set password_hash = $1 where id = $2`
	_, err := r.dbConn.Exec(query, passHash, id)
	return err
}

func (r *authRepo) SetVerified(userId uuid.UUID) error {
	query := `update users set is_verified = true where id = $1`
	_, err := r.dbConn.Exec(query, userId)
	return err
}

func (r *authRepo) Upgrade(id string, user auth.User) (auth.User, error) {
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
	err := r.dbConn.QueryRowx(query,
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
	dbConn *sqlx.DB
}

func NewVerifierRepo(
	dbConn *sqlx.DB,
) auth.VerifierRepo {
	return &verifierRepo{
		dbConn: dbConn,
	}
}

func (r *verifierRepo) Create(verifier auth.Verifier) error {
	query := `insert into 
		verifier(user_id, token_hash)
		values(:user_id, :token_hash)
	`

	_, err := r.dbConn.NamedExec(query, verifier)
	return err
}

func (r *verifierRepo) GetByHash(tokenHash string) (auth.Verifier, error) {
	query := `select * from verifier where token_hash = $1 and expire_at > now()`
	var verifier auth.Verifier
	if err := r.dbConn.Get(&verifier, query, tokenHash); err != nil {
		return auth.Verifier{}, err
	}
	return verifier, nil
}

func (r *verifierRepo) GetById(userId uuid.UUID) (auth.Verifier, error) {
	query := `select * from verifier where user_id = $1`
	var verifier auth.Verifier
	if err := r.dbConn.Get(&verifier, query, userId); err != nil {
		return auth.Verifier{}, err
	}
	return verifier, nil
}

func (r *verifierRepo) Delete(id int64) error {
	query := `delete from verifier where id = $1`
	_, err := r.dbConn.Exec(query, id)
	return err
}

type reseterRepo struct {
	dbConn *sqlx.DB
}

func NewReseterRepo(
	dbConn *sqlx.DB,
) auth.ReseterRepo {
	return &reseterRepo{
		dbConn: dbConn,
	}
}

func (r *reseterRepo) GetById(id uuid.UUID) (auth.Reseter, error) {
	query := `select * from reseter where user_id = $1 and expire_at > now()`
	var reseter auth.Reseter
	if err := r.dbConn.Get(&reseter, query, id); err != nil {
		return auth.Reseter{}, err
	}
	return reseter, nil
}

// no use case
func (r *reseterRepo) Update(reseter auth.Reseter) error {
	return nil
}

func (r *reseterRepo) Create(reseter auth.Reseter) error {
	query := `insert into 
		reseter(user_id, token_hash)
		values(:user_id, :token_hash)
	`

	_, err := r.dbConn.NamedExec(query, reseter)
	return err
}

func (r *reseterRepo) GetByToken(token string) (auth.Reseter, error) {
	query := `select * from reseter where token_hash = $1 and expire_at > now()`
	var reseter auth.Reseter
	if err := r.dbConn.Get(&reseter, query, token); err != nil {
		return auth.Reseter{}, err
	}
	return reseter, nil
}

func (r *reseterRepo) DeleteById(id int64) error {
	query := `delete from reseter where id = $1`
	_, err := r.dbConn.Exec(query, id)
	return err
}
