# LINE Chatbot

This repository contains the source code of a Line chatbot that reply to user messages or group chats with Google Gemini AI.

## Running

`cmd/lambda/main.go` is for AWS Lambda runtime.

`cmd/server/main.go` is for local runtime.

## Deploying

For deploying to AWS Lambda, please refer to [AWS Documents](https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html).

For running with docker, please use `vgjm/linebot` docker image.