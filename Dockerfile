FROM golang:1.10

ENV KUBE_LATEST_VERSION="v1.10.0"

RUN apt-get update \
 && apt-get install -y curl net-tools \
 && curl -L https://storage.googleapis.com/kubernetes-release/release/${KUBE_LATEST_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
 && chmod +x /usr/local/bin/kubectl

ADD metric-scraper.go /go/src/metric-scraper/metric-scraper.go

RUN go get metric-scraper

ENTRYPOINT ["go", "run", "/go/src/metric-scraper/metric-scraper.go"]
