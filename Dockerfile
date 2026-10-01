FROM golang:1.27.1 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# distroless/static has no libc, so the binary must not link against one.
RUN CGO_ENABLED=0 go build -o /out/metric-scraper .

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/metric-scraper /metric-scraper
# The nonroot user by number: a pod with runAsNonRoot can only verify a numeric user.
USER 65532:65532
CMD [ "/metric-scraper" ]
