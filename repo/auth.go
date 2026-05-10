package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

var ctx = context.Background()

type AuthRepository interface {
	GetByEmail(email string) (model.User, error)
	GetById(id uuid.UUID) (model.User, error)
	Create(user model.User) (model.User, error)
	DeleteById(id uuid.UUID) error
	DeleteByEmail(email string) error
	UpdatePassword(id uuid.UUID, passHash string) error
	SetVerified(userId uuid.UUID) error
	Upgrade(id string, user model.User) (model.User, error)
	CreateDemo(user model.AnonUser) (model.AnonUser, error)
}

type authRepo struct {
	dbConn *sqlx.DB
}

func NewAuthRepository(
	dbConn *sqlx.DB,
) AuthRepository {
	return &authRepo{
		dbConn: dbConn,
	}
}

func (r *authRepo) GetByEmail(email string) (model.User, error) {
	query := `select * from users where email = $1`
	var user model.User
	if err := r.dbConn.Get(&user, query, email); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (r *authRepo) GetById(id uuid.UUID) (model.User, error) {
	query := `select * from users where id = $1`
	var user model.User
	if err := r.dbConn.Get(&user, query, id); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (r *authRepo) Create(user model.User) (model.User, error) {
	query := `insert into 
		users(username, fullname, email, password_hash, is_verified, role, deleted_at)
		values(:username, :fullname, :email, :password_hash, :is_verified, :role, :deleted_at)
		returning id, username, fullname, email, is_verified, role, created_at
	`

	rows, err := r.dbConn.NamedQuery(query, user)
	if err != nil {
		return model.User{}, err
	}
	defer rows.Close()

	var created model.User
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return model.User{}, err
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

func (r *authRepo) Upgrade(id string, user model.User) (model.User, error) {
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

	var updated model.User
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
		return model.User{}, err
	}

	return updated, nil
}

func (r *authRepo) CreateDemo(user model.AnonUser) (model.AnonUser, error) {
	query := `insert into 
		anon_users(username, fullname)
		values(:username, :fullname)
		returning id, username, fullname, created_at
	`

	rows, err := r.dbConn.NamedQuery(query, user)
	if err != nil {
		return model.AnonUser{}, err
	}
	defer rows.Close()

	var created model.AnonUser
	if rows.Next() {
		if err := rows.StructScan(&created); err != nil {
			return model.AnonUser{}, err
		}
	}
	return created, nil
}
