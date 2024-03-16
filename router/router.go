package router

import (
	"strings"
)

type MessageCase string

const (
	MENU     MessageCase = "Check the menu"
	AI_REPLY MessageCase = "Use AI to generate response"
)

const (
	NO_REPLY MessageCase = "No need to reply"
)

func Route(text string) MessageCase {
	if strings.HasPrefix(text, "/") {
		switch text {
		case "/菜单":
			return MENU
		default:
			return AI_REPLY
		}
	}
	return NO_REPLY
}
