#!/bin/bash

# 确保 CN （逐步淘汰） 和 SANs 匹配 AA Server 的完全限定域名（FQDN）
CN="simple-aa-server.aa-system.svc"

certsPath="certs/"

mkdir ${certsPath}

# 生成 KEY 和 CSR ，保存为 tls.csr 和 tls.key
openssl req -newkey rsa:2048 -nodes -keyout ${certsPath}tls.key -out ${certsPath}tls.csr -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=${CN}"

# 生成 10 年有效期的自签名根证书作为 CA ，保存为 ca.crt 和 ca.key
openssl req -new -x509 -days 3650 -nodes -out ${certsPath}ca.crt -keyout ${certsPath}ca.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=Acme Root CA"
# 使用 CA 和 CSR 签发 10 年有效期的 CRT 证书，保存为 tls.crt
openssl x509 -req -days 3650 -in ${certsPath}tls.csr -CA ${certsPath}ca.crt -CAkey ${certsPath}ca.key -CAcreateserial -out ${certsPath}tls.crt -extfile <(printf "subjectAltName=DNS:${CN}")


# 使用 base64 编码证书和密钥，去除换行符
ca_cert=$(cat ${certsPath}ca.crt | base64 | tr -d '\n')
tls_cert=$(cat ${certsPath}tls.crt | base64 | tr -d '\n')
tls_key=$(cat ${certsPath}tls.key | base64 | tr -d '\n')

# 替换 deploy.yaml 文件中的占位符
sed -i "s|<base64-encoded-ca-cert>|$ca_cert|g" deploy.yaml
sed -i "s|<base64-encoded-tls-cert>|$tls_cert|g" deploy.yaml
sed -i "s|<base64-encoded-tls-key>|$tls_key|g" deploy.yaml
