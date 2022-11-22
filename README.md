# Build application

- `cmd /V /C "set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -o app main.go"`
- `CGO_ENABLED=0 GOOS=linux GOOS=linux GOARCH=amd64 go build -o app main.go`

# Build Image and push to gcr

```
docker build . -t gcr.io/stream-capture/stream-capture
docker push gcr.io/stream-capture/stream-capture
```

# Deploy

```
gcloud app deploy --image-url=gcr.io/stream-capture/stream-capture --stop-previous-version .\app.yml
```

# Running

```
docker run -p 8080:8080 --env "STREAM_URL=https://some-url" --env "CRON=0 14 55 * * *" --env "REC_TIMEOUT=900" gcr.io/stream-capture/stream-capture --env "FTP=user:password@domain/path/x/y/z"
```
