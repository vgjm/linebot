package storage

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GroupUserSetting struct {
	GroupId           string `dynamodbav:"GroupId"`
	UserId            string `dynamodbav:"UserId"`
	SystemInstruction string `dynamodbav:"SystemInstruction"`
}

func (setting GroupUserSetting) GetKey() map[string]types.AttributeValue {
	gid, err := attributevalue.Marshal(setting.GroupId)
	if err != nil {
		panic(err)
	}
	uid, err := attributevalue.Marshal(setting.UserId)
	if err != nil {
		panic(err)
	}
	return map[string]types.AttributeValue{"GroupId": gid, "UserId": uid}
}
