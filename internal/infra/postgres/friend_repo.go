package postgres

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ffgif/internal/domain/friend"
)

type friendRepo struct {
	db *sqlx.DB
}

func NewFriendRepository(db *sqlx.DB) friend.Repository {
	return &friendRepo{db: db}
}

func (r *friendRepo) Create(ctx context.Context, requesterID, addresseeID string) (friend.Friendship, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		insert into friendships(requester_id, addressee_id, status)
		values ($1, $2, 'pending')
		returning id, requester_id, addressee_id, status, created_at, updated_at
	`
	var f friend.Friendship
	if err := sqlx.GetContext(ctx, db, &f, query, requesterID, addresseeID); err != nil {
		return friend.Friendship{}, err
	}
	return f, nil
}

func (r *friendRepo) Exists(ctx context.Context, userA, userB string) (bool, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select exists(
			select 1 from friendships
			where (requester_id = $1 and addressee_id = $2)
			   or (requester_id = $2 and addressee_id = $1)
		)
	`
	var exists bool
	if err := sqlx.GetContext(ctx, db, &exists, query, userA, userB); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *friendRepo) GetByID(ctx context.Context, id string) (friend.Friendship, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `select id, requester_id, addressee_id, status, created_at, updated_at from friendships where id = $1`
	var f friend.Friendship
	if err := sqlx.GetContext(ctx, db, &f, query, id); err != nil {
		return friend.Friendship{}, err
	}
	return f, nil
}

func (r *friendRepo) UpdateStatus(ctx context.Context, id string, status string, actingUserID string) (friend.Friendship, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		update friendships
		set status = $1, updated_at = now()
		where id = $2 and addressee_id = $3
		returning id, requester_id, addressee_id, status, created_at, updated_at
	`
	var f friend.Friendship
	if err := sqlx.GetContext(ctx, db, &f, query, status, id, actingUserID); err != nil {
		return friend.Friendship{}, err
	}
	return f, nil
}

func (r *friendRepo) Delete(ctx context.Context, id string, actingUserID string) error {
	db := getDBFromCtx(ctx, r.db)
	query := `
		delete from friendships
		where id = $1 and (requester_id = $2 or addressee_id = $2)
	`
	_, err := db.ExecContext(ctx, query, id, actingUserID)
	return err
}

func (r *friendRepo) ListFriends(ctx context.Context, userID string) ([]friend.FriendResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select
			u.id as user_id, u.username, u.fullname, f.status, f.created_at
		from friendships f
		join users u on u.id = case
			when f.requester_id = $1 then f.addressee_id
			else f.requester_id
		end
		where (f.requester_id = $1 or f.addressee_id = $1)
		  and f.status = 'accepted'
	`
	var results []friend.FriendResponse
	if err := sqlx.SelectContext(ctx, db, &results, query, userID); err != nil {
		return []friend.FriendResponse{}, err
	}
	return results, nil
}

func (r *friendRepo) ListPendingIncoming(ctx context.Context, userID string) ([]friend.FriendResponse, error) {
	db := getDBFromCtx(ctx, r.db)
	query := `
		select
			u.id as user_id, u.username, u.fullname, f.status, f.created_at
		from friendships f
		join users u on u.id = f.requester_id
		where f.addressee_id = $1 and f.status = 'pending'
	`
	var results []friend.FriendResponse
	if err := sqlx.SelectContext(ctx, db, &results, query, userID); err != nil {
		return []friend.FriendResponse{}, err
	}
	return results, nil
}
