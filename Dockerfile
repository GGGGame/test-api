# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Imposta la directory di lavoro
WORKDIR /app

# Copia il file sorgente (assumendo che non ci siano dipendenze esterne complesse)
# Se hai un file go.mod, copia anche quello prima di 'go build'
COPY mock-backend.go .

# Compila l'eseguibile statico per massime prestazioni
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mock-server mock-backend.go

# Stage 2: Runtime (Immagine finale leggerissima)
FROM scratch

# Copia solo l'eseguibile compilato dallo stage precedente
COPY --from=builder /app/mock-server /mock-server

# Espone la porta 4000
EXPOSE 4000

# Avvia il server
ENTRYPOINT ["/mock-server"]