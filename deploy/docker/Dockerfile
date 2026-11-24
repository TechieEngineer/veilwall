FROM node:22-bookworm-slim AS frontend
WORKDIR /src/web
COPY web/ ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi && npm run build
FROM golang:1.25-bookworm AS backend
WORKDIR /src
COPY . .
COPY --from=frontend /src/web/dist/ ./web/dist/
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -o /out/veilwall ./cmd/veilwall
FROM debian:bookworm-slim
COPY --from=backend /out/veilwall /usr/local/bin/veilwall
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/veilwall"]
CMD ["--demo","--bind=127.0.0.1","--port=8080"]
