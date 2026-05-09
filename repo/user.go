package repo

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/model"
)

type UserRepository interface {
	GetProfile(id string) (model.ProfileResp, error)
	SetProfile(profile model.Profile) error
	UpdateProfile(profile model.ProfileResp, userId string) (model.ProfileResp, error)
	ChangePassword(userId string, hash string) error
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

func (r *userRepo) UpdateProfile(profile model.ProfileResp, userId string) (model.ProfileResp, error) {
	query1 := `
	update users 
	set
		username = COALESCE($1, username),
		fullname = COALESCE($2, fullname),
		updated_at = NOW()
	where id = $3
	`
	query2 := `
	update profiles
	set
		profile_pic = COALESCE($1, profile_pic),
		updated_at = NOW()
	where user_id = $2
	`

	_, err := r.dbConn.Exec(query1, profile.Username, profile.Fullname, userId)
	if err != nil {
		return model.ProfileResp{}, err
	}

	_, err = r.dbConn.Exec(query2, profile.ProfilePic, userId)
	if err != nil {
		return model.ProfileResp{}, err
	}

	return r.GetProfile(userId)
}

func (r *userRepo) ChangePassword(userId string, hash string) error {
	query1 := `
	update users 
	set
		password_hash = $1,
		updated_at = NOW()
	where id = $2
	`
	_, err := r.dbConn.Exec(query1, hash, userId)
	if err != nil {
		return err
	}
	return nil
}
