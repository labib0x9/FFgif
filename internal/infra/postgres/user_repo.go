package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/user"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) user.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) GetProfile(ctx context.Context, id string, forUpdate bool) (user.ProfileResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select
			u.username, p.profile_pic, u.fullname, u.email, u.is_verified, u.updated_at
		from profiles p
		left join users u
		on
			u.id = p.user_id
		where
		 	u.id = $1
	`
	if forUpdate {
		query += "for update"
	}
	var profile user.ProfileResponse
	if err := sqlx.GetContext(ctx, db, &profile, query, id); err != nil {
		return user.ProfileResponse{}, err
	}
	return profile, nil
}

func (r *userRepo) SetProfile(ctx context.Context, profile user.Profile) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		profiles(user_id, profile_pic)
		values(:user_id, :profile_pic)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, profile)
	return err
}

func (r *userRepo) UpdateProfile(ctx context.Context, req user.ProfileUpdateRequest, userId string) (user.ProfileResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		WITH updated_user AS (
			UPDATE users
			SET
				username = COALESCE($1, username),
				fullname = COALESCE($2, fullname),
				updated_at = NOW()
			WHERE id = $3
			RETURNING id, username, fullname, email, is_verified, updated_at
		),
		updated_profile AS (
			UPDATE profiles
			SET
				profile_pic = COALESCE($4, profile_pic),
				updated_at = NOW()
			WHERE user_id = $3
			RETURNING user_id, profile_pic
		)
		SELECT u.username, u.fullname, u.email, u.is_verified, p.profile_pic, u.updated_at
		FROM updated_user u
		LEFT JOIN updated_profile p ON p.user_id = u.id
	`

	var resp user.ProfileResponse
	err := sqlx.GetContext(ctx, db, &resp, query, req.Username, req.Fullname, userId, req.ProfilePic)
	if err != nil {
		return user.ProfileResponse{}, err
	}
	return resp, nil
}

func (r *userRepo) ChangePassword(ctx context.Context, userId string, hash string) error {
	db := getDBFromCtx(ctx, r.db)
	query1 := `
	update users 
	set
		password_hash = $1,
		updated_at = NOW()
	where id = $2
	`
	_, err := db.ExecContext(ctx, query1, hash, userId)
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
	db *sqlx.DB
}

func NewAnonUserRepository(db *sqlx.DB) user.AnonUserRepository {
	return &anonUserRepo{db: db}
}

func (r *anonUserRepo) GetProfile(ctx context.Context, id string) (user.ProfileResponse, error) {
	db := getDBFromCtx(ctx, r.db)
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
	var profile user.ProfileResponse
	if err := sqlx.GetContext(ctx, db, &profile, query, id); err != nil {
		return user.ProfileResponse{}, err
	}
	return profile, nil
}

func (r *anonUserRepo) SetProfile(ctx context.Context, profile user.Profile) error {
	db := getDBFromCtx(ctx, r.db)
	query := `insert into 
		anon_profiles(user_id, profile_pic)
		values(:user_id, :profile_pic)
	`

	_, err := sqlx.NamedExecContext(ctx, db, query, profile)
	return err
}
