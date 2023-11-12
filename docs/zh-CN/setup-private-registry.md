部署私有 Docker 仓库
====================

1. 创建 auth 目录

```bash
$ mkdir auth
```
2. 创建登录密码

```bash
$ sudo apt install apache2-utils -y
$ htpasswd -Bc registry.password [username]
```

3. 创建 `docker-compose.yml` 文件

内容如下：

```yaml
version: "3.1"
services:
  traefik:
    image: traefik:v2.3
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--providers.docker.exposedbydefault=false"
      - "--certificatesresolvers.myresolver.acme.httpchallenge=true"
      - "--certificatesresolvers.myresolver.acme.httpchallenge.entrypoint=web"
      - "--certificatesresolvers.myresolver.acme.email=daqing@mzevo.com"
      - "--certificatesresolvers.myresolver.acme.storage=/letsencrypt/acme.json"
    ports:
      - 80:80
      - 443:443
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ssl:/letsencrypt

  registry:
    image: "registry:2"
    restart: "always"
    environment:
      REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY: /data
      REGISTRY_AUTH: htpasswd
      REGISTRY_AUTH_HTPASSWD_REALM: Registry
      REGISTRY_AUTH_HTPASSWD_PATH: /auth/registry.password
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.reg.rule=Host(`reg.mzcall.com`)"
      - "traefik.http.routers.reg.entrypoints=websecure"
      - "traefik.http.routers.reg.tls.certresolver=myresolver"
      - "traefik.http.services.reg.loadbalancer.server.port=5000"
    volumes:
      - regdata:/data
      - ./auth:/auth

volumes:
  ssl:
  regdata:
```

