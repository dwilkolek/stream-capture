# Build application

- `cmd /V /C "set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -o app main.go"`
- `CGO_ENABLED=0 GOOS=linux GOOS=linux GOARCH=amd64 go build -o app main.go`

# Build Image and push to gcr

```
docker build . -t gcr.io/stream-capture/stream-capture
docker push gcr.io/stream-capture/stream-capture
```

# Deploy image from gcr

```
gcloud app deploy --image-url=gcr.io/stream-capture/stream-capture --stop-previous-version .\app.yml
```

# Deploy to go116

```
gcloud app deploy --stop-previous-version .\app.yml
```

# Running

```
docker run -p 8080:8080 --env "STREAM_URL=https://some-url" --env "CRON=0 14 55 * * *" --env "REC_TIMEOUT=900" --env "FTP=user:password@domain/path/x/y/z" gcr.io/stream-capture/stream-capture
```

```
go run main.go './ffmpeg' 'https://some-url' '0 14 55 * * *' '900' 'user:password@domain/path/x/y/z'
```
