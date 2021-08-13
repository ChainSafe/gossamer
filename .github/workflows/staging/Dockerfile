FROM golang:1.16

ARG chain="polkadot"
ARG basepath="~/.gossamer"

ENV GO111MODULE=on

ENV chain=${chain}
ENV basepath=${basepath}

RUN mkdir /gossamer
WORKDIR /gossamer

COPY . .

RUN go get ./...
RUN go build github.com/ChainSafe/gossamer/cmd/gossamer

RUN ["sh", "-c", "gossamer init --chain=${chain}"]
ENTRYPOINT ["sh", "-c", "gossamer --chain=${chain} --basepath=${basepath}/${chain}"]
EXPOSE 7001 8546 8540