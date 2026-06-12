#!/bin/bash
set -euo pipefail
exec > /var/log/bootstrap.log 2>&1

echo "=== Starting bootstrap ==="

apt-get update -y
apt-get install -y ca-certificates curl gnupg nginx apache2-utils git make

# Docker
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update -y
apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

usermod -aG docker ubuntu
systemctl enable docker
systemctl start docker

# Basic auth credentials
htpasswd -cb /etc/nginx/.htpasswd "${nginx_username}" "${nginx_password}"

# Nginx - single server block, each UI gets a path prefix
cat > /etc/nginx/sites-available/pipeline << 'EOF'
server {
    listen 8080;
    server_name _;

    auth_basic "Pipeline";
    auth_basic_user_file /etc/nginx/.htpasswd;

    location / {
        proxy_pass http://localhost:8088/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }

    # add more locations here later e.g.
    # location /jaeger/ {
    #     proxy_pass http://localhost:16686/;
    # }
}
EOF

ln -sf /etc/nginx/sites-available/pipeline /etc/nginx/sites-enabled/pipeline
rm -f /etc/nginx/sites-enabled/default

nginx -t
systemctl enable nginx
systemctl restart nginx

echo "=== Bootstrap complete ==="
