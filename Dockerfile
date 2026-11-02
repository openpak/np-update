FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/np-update .

FROM alpine:3.22
LABEL org.opencontainers.image.source="https://github.com/openpak/np-update"
RUN addgroup -S app && adduser -S -G app app
USER app
COPY --from=build /out/np-update /usr/local/bin/np-update
EXPOSE 8534
ENTRYPOINT ["/usr/local/bin/np-update"]
