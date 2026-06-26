package postgres

import (
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

type userRepo struct {
	dbConn *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) user.UserRepository {
	return &userRepo{dbConn: db}
}

func (r *userRepo) GetProfile(id string) (user.ProfileResp, error) {
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
	var profile user.ProfileResp
	if err := r.dbConn.Get(&profile, query, id); err != nil {
		return user.ProfileResp{}, err
	}
	return profile, nil
}

func (r *userRepo) SetProfile(profile user.Profile) error {
	query := `insert into 
		profiles(user_id, profile_pic)
		values(:user_id, :profile_pic)
	`

	_, err := r.dbConn.NamedExec(query, profile)
	return err
}

func (r *userRepo) UpdateProfile(profile user.ProfileResp, userId string) (user.ProfileResp, error) {
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
		return user.ProfileResp{}, err
	}

	_, err = r.dbConn.Exec(query2, profile.ProfilePic, userId)
	if err != nil {
		return user.ProfileResp{}, err
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

// type anonAuthRepo struct {
// 	dbConn *sqlx.DB
// }

// func NewAnonAuthRepository(
// 	dbConn *sqlx.DB,
// ) user.AnonAuthRepository {
// 	return &anonAuthRepo{
// 		dbConn: dbConn,
// 	}
// }

// func (r *anonAuthRepo) GetById(id uuid.UUID) (user.AnonUser, error) {
// 	query := `select * from anon_users where id = $1`
// 	var user user.AnonUser
// 	if err := r.dbConn.Get(&user, query, id); err != nil {
// 		return user.AnonUser{}, err
// 	}
// 	return user, nil
// }

// func (r *anonAuthRepo) Create(user user.AnonUser) (user.AnonUser, error) {
// 	query := `insert into
// 		anon_users(username, fullname)
// 		values(:username, :fullname)
// 		returning id, username, fullname, created_at
// 	`

// 	rows, err := r.dbConn.NamedQuery(query, user)
// 	if err != nil {
// 		return user.AnonUser{}, err
// 	}
// 	defer rows.Close()

// 	var created user.AnonUser
// 	if rows.Next() {
// 		if err := rows.StructScan(&created); err != nil {
// 			return user.AnonUser{}, err
// 		}
// 	}
// 	return created, nil
// }

type anonUserRepo struct {
	dbConn *sqlx.DB
}

func NewAnonUserRepository(db *sqlx.DB) user.AnonUserRepository {
	return &anonUserRepo{dbConn: db}
}

func (r *anonUserRepo) GetProfile(id string) (user.ProfileResp, error) {
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
	var profile user.ProfileResp
	if err := r.dbConn.Get(&profile, query, id); err != nil {
		return user.ProfileResp{}, err
	}
	return profile, nil
}

func (r *anonUserRepo) SetProfile(profile user.Profile) error {
	query := `insert into 
		anon_profiles(user_id, profile_pic)
		values(:user_id, :profile_pic)
	`

	_, err := r.dbConn.NamedExec(query, profile)
	return err
}
