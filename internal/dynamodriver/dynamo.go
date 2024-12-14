package dynamodriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vgjm/linebot/internal/storage"
)

type DynamoDriver struct {
	client *dynamodb.Client
}

func New(ctx context.Context) (*DynamoDriver, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg)

	driver := &DynamoDriver{client}

	if err := driver.init(ctx); err != nil {
		return nil, err
	}

	return driver, nil
}

func (d *DynamoDriver) init(ctx context.Context) error {
	if err := d.createGroupUserSettingTableIfNotExist(ctx); err != nil {
		return err
	}
	if err := d.createUserSettingTableIfNotExist(ctx); err != nil {
		return err
	}
	return nil
}

func (d *DynamoDriver) determinTableExist(ctx context.Context, name string) (bool, error) {
	exists := true
	_, err := d.client.DescribeTable(
		ctx, &dynamodb.DescribeTableInput{TableName: aws.String(name)},
	)
	if err != nil {
		var notFoundEx *types.ResourceNotFoundException
		if !errors.As(err, &notFoundEx) {
			return false, err
		}
		exists = false
	}
	return exists, nil
}

func (d *DynamoDriver) createTableAndWait(ctx context.Context, input *dynamodb.CreateTableInput) error {
	tablename := *input.TableName
	exists, err := d.determinTableExist(ctx, tablename)
	if err != nil {
		return fmt.Errorf("couldn't determine existence of table %v. Error: %w", tablename, err)
	}
	if !exists {
		slog.Info("a table does not exist. creating it...", "table", tablename)
		_, err := d.client.CreateTable(ctx, input)
		if err != nil {
			return fmt.Errorf("couldn't create table %v. Error: %w", tablename, err)
		} else {
			waiter := dynamodb.NewTableExistsWaiter(d.client)
			err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(tablename)}, 5*time.Minute)
			if err != nil {
				return fmt.Errorf("couldn't create table %v. Error: %w", tablename, err)
			}
			slog.Info("table created successfully.", "table", tablename)
		}
	}
	return nil
}

func (d *DynamoDriver) createUserSettingTableIfNotExist(ctx context.Context) error {
	return d.createTableAndWait(ctx, &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String("UserId"),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String("UserId"),
			KeyType:       types.KeyTypeHash,
		}},
		TableName: aws.String(storage.UserSettingTableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
}

func (d *DynamoDriver) createGroupUserSettingTableIfNotExist(ctx context.Context) error {
	return d.createTableAndWait(ctx, &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{{
			AttributeName: aws.String("GroupId"),
			AttributeType: types.ScalarAttributeTypeS,
		}, {
			AttributeName: aws.String("UserId"),
			AttributeType: types.ScalarAttributeTypeS,
		}},
		KeySchema: []types.KeySchemaElement{{
			AttributeName: aws.String("GroupId"),
			KeyType:       types.KeyTypeHash,
		}, {
			AttributeName: aws.String("UserId"),
			KeyType:       types.KeyTypeRange,
		}},
		TableName: aws.String(storage.GroupUserSettingTableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	})
}

func (d *DynamoDriver) UpsertGroupUserSetting(ctx context.Context, setting storage.GroupUserSetting) error {
	var response *dynamodb.UpdateItemOutput
	var attribute map[string]string
	update := expression.Set(expression.Name(storage.SystemInstruction), expression.Value(setting.SystemInstruction))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return err
	} else {
		response, err = d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName:                 aws.String(storage.GroupUserSettingTableName),
			Key:                       setting.GetKey(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			UpdateExpression:          expr.Update(),
			ReturnValues:              types.ReturnValueUpdatedNew,
		})
		if err != nil {
			return err
		} else {
			err = attributevalue.UnmarshalMap(response.Attributes, &attribute)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DynamoDriver) GetGroupUserSetting(ctx context.Context, groupId, userId string) (*storage.GroupUserSetting, error) {
	guSetting := storage.GroupUserSetting{GroupId: groupId, UserId: userId}
	response, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       guSetting.GetKey(),
		TableName: aws.String(storage.GroupUserSettingTableName),
	})
	if err != nil {
		return nil, err
	} else {
		err = attributevalue.UnmarshalMap(response.Item, &guSetting)
		if err != nil {
			return nil, err
		}
	}
	return &guSetting, nil
}

func (d *DynamoDriver) UpsertUserSetting(ctx context.Context, setting storage.UserSetting) error {
	var response *dynamodb.UpdateItemOutput
	var attribute map[string]string
	update := expression.Set(expression.Name(storage.SystemInstruction), expression.Value(setting.SystemInstruction))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return err
	} else {
		response, err = d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName:                 aws.String(storage.UserSettingTableName),
			Key:                       setting.GetKey(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			UpdateExpression:          expr.Update(),
			ReturnValues:              types.ReturnValueUpdatedNew,
		})
		if err != nil {
			return err
		} else {
			err = attributevalue.UnmarshalMap(response.Attributes, &attribute)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *DynamoDriver) GetUserSetting(ctx context.Context, userId string) (*storage.UserSetting, error) {
	uSetting := storage.UserSetting{UserId: userId}
	response, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       uSetting.GetKey(),
		TableName: aws.String(storage.UserSettingTableName),
	})
	if err != nil {
		return nil, err
	} else {
		err = attributevalue.UnmarshalMap(response.Item, &uSetting)
		if err != nil {
			return nil, err
		}
	}
	return &uSetting, nil
}
