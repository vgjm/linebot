package storage

import "context"

type Storage interface {
	UpsertGroupUserSetting(ctx context.Context, setting GroupUserSetting) error
	GetGroupUserSetting(ctx context.Context, groupId, userId string) (*GroupUserSetting, error)
	UpsertUserSetting(ctx context.Context, setting UserSetting) error
	GetUserSetting(ctx context.Context, userId string) (*UserSetting, error)
}
