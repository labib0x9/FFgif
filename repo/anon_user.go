package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type AnonUserRepository interface {
	GetProfile(id string) (model.ProfileResp, error)
	SetProfile(profile model.Profile) error
}

type anonUserRepo struct {
	dbConn *sqlx.DB
}

func NewAnonUserRepository(db *sqlx.DB) AnonUserRepository {
	return &anonUserRepo{dbConn: db}
}

func (r *anonUserRepo) GetProfile(id string) (model.ProfileResp, error) {
	query := `
		select
			u.username, p.profile_pic, u.fullname
		from anon_profiles p
		left join anon_users u
		on
			u.id = p.user_id
		where
		 	u.id = $1
	`
	var profile model.ProfileResp
	if err := r.dbConn.Get(&profile, query, id); err != nil {
		return model.ProfileResp{}, err
	}
	return profile, nil
}

func (r *anonUserRepo) SetProfile(profile model.Profile) error {
	query := `insert into 
		anon_profiles(user_id, profile_pic)
		values(:user_id, :profile_pic)
	`

	_, err := r.dbConn.NamedExec(query, profile)
	return err
}
