FROM alpine:3.18 as certs
RUN apk --no-cache add ca-certificates

FROM scratch
# 复制证书
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# 复制二进制
COPY main /main
ENTRYPOINT ["/main"]
