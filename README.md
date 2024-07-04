## LINE Chat Bot

This repository is for a simple LINE chat bot.

```
$env:GOOS = "linux"
$env:GOARCH = "arm64"
$env:CGO_ENABLED = "0"

go build -tags lambda.norpc -o bootstrap main.go

~\go\bin\build-lambda-zip.exe -o linebotFunction.zip .\bootstrap
2024/07/04 23:09:44 wrote linebotFunction.zip

aws lambda update-function-code --function-name linebotFunction --zip-file fileb://linebotFunction.zip
```