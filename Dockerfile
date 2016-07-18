FROM golang:1.6.2-alpine

COPY . $GOPATH/src/github.com/dev/indexer
WORKDIR $GOPATH/src/github.com/dev/indexer
RUN go build -o indexer github.com/dev/indexer/cmd/server && \
    mv indexer $GOPATH/bin
EXPOSE 8080
ENTRYPOINT ["indexer"] 
