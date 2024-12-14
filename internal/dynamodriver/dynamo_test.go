package dynamodriver

import (
	"context"
	"testing"

	"github.com/vgjm/linebot/internal/storage"
)

func TestDynamoDriver(t *testing.T) {
	driver, err := New(context.TODO())
	if err != nil {
		t.Fatalf("failed to initialize dynamo driver client: %v\n", err)
	}

	ctx := context.TODO()
	instruct := ""
	testGroupId := "test"
	testUserId := "test"
	if err := driver.UpsertGroupUserSetting(ctx, storage.GroupUserSetting{
		GroupId:           testGroupId,
		UserId:            testUserId,
		SystemInstruction: instruct,
	}); err != nil {
		t.Fatalf("failed to update group user setting: %v\n", err)
	}
	setting, err := driver.GetGroupUserSetting(ctx, testGroupId, testUserId)
	if err != nil {
		t.Fatalf("failed to get group user setting: %v\n", err)
	}
	if setting.SystemInstruction != instruct {
		t.Fatalf("got different system instruction, got: %v, expect: %v\n", setting.SystemInstruction, instruct)
	}

	if err := driver.UpsertUserSetting(ctx, storage.UserSetting{
		UserId:            testUserId,
		SystemInstruction: instruct,
	}); err != nil {
		t.Fatalf("failed to update user setting: %v\n", err)
	}
	uSetting, err := driver.GetUserSetting(ctx, testUserId)
	if err != nil {
		t.Fatalf("failed to get user setting: %v\n", err)
	}
	if uSetting.SystemInstruction != instruct {
		t.Fatalf("got different system instruction, got: %v, expect: %v\n", setting.SystemInstruction, instruct)
	}
}
