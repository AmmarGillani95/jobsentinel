# SlinkedIn
An AWS lambda to hit LinkedIn job results and email me any relevant listings posted in the last 10 minutes

### build (linux):
```
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap main.go
zip slinkedin.zip bootstrap
```
