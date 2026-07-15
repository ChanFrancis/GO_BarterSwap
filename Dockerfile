# Étape de build : compile le binaire statiquement
FROM golang:1.26 AS build

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /barterswap .

# Étape finale : image minimale, sans shell ni root
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /barterswap /barterswap

EXPOSE 8080
ENTRYPOINT ["/barterswap"]
