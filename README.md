# LINE Chatbot

This repository contains the source code of a Line chatbot that reply to user messages or group chats with Google Gemini AI.

## Deploy to AWS Lambda

Set Environments (Powershell on Windows):

```
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"
```

Build the application

```
go build -tags lambda.norpc -o bootstrap main.go
```

Archive

```
~\go\bin\build-lambda-zip.exe -o linebotFunction.zip .\bootstrap
2024/07/04 23:09:44 wrote linebotFunction.zip
```

Deploy

```
aws lambda update-function-code --function-name linebotFunction --zip-file fileb://linebotFunction.zip
```