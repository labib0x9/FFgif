package repo

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type AnonAuthRepository interface {
	GetById(id uuid.UUID) (model.AnonUser, error)
	Create(user model.AnonUser) (model.AnonUser, error)
}

type anonAuthRepo struct {
	dbConn *sqlx.DB
}

func NewAnonAuthRepository(
	dbConn *sqlx.DB,
) AnonAuthRepository {
	return &anonAuthRepo{
		dbConn: dbConn,
	}
}

func (r *anonAuthRepo) GetById(id uuid.UUID) (model.AnonUser, error) {
	query := `select * from anon_users where id = $1`
	var user model.AnonUser
	if err := r.dbConn.Get(&user, query, id); err != nil {
		return model.AnonUser{}, err
	}
	return user, nil
}

func (r *anonAuthRepo) Create(user model.AnonUser) (model.AnonUser, error) {
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
