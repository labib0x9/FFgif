package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type UserRepository interface {
	GetProfile(id string) (model.ProfileResp, error)
	SetProfile(profile model.Profile) error
}

type userRepo struct {
	dbConn *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepo{dbConn: db}
}

func (r *userRepo) GetProfile(id string) (model.ProfileResp, error) {
	query := `
		select
			u.username, p.profile_pic, u.fullname, u.email, u.is_verified
		from profiles p
		left join users u
		on
			u.id = p.user_id
		where
		 	u.id = $1
	`
	// query := `select * from profiles where user_id = $1`
	var profile model.ProfileResp
	if err := r.dbConn.Get(&profile, query, id); err != nil {
		return model.ProfileResp{}, err
	}
	return profile, nil
}

func (r *userRepo) SetProfile(profile model.Profile) error {
	query := `insert into 
		profiles(user_id, profile_pic)
		values(:user_id, :profile_pic)
	`

	_, err := r.dbConn.NamedExec(query, profile)
	return err
}
