package storage

import (
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UserSetting struct {
	UserId            string `dynamodbav:"UserId"`
	SystemInstruction string `dynamodbav:"SystemInstruction"`
}

func (setting UserSetting) GetKey() map[string]types.AttributeValue {
	uid, err := attributevalue.Marshal(setting.UserId)
	if err != nil {
		panic(err)
	}
	return map[string]types.AttributeValue{"UserId": uid}
}
