# units — typed unit-calculation HTTP service
FROM golang:1.27-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/units ./cmd/units

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/units /usr/local/bin/units
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/units"]
CMD ["-addr", ":8080"]
