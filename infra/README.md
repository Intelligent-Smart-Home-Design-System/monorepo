## Terraform конфиг для инфраструктуры веб-приложения и пайплайна построения каталога

Работает с yandex cloud.  
Поднимает vm с параметрами:
- Ubuntu 22.04
- CPU: 4 cores
- RAM: 8 GB
- Disk: 50 GB SSD
- Зона: ru-central1-a

Устанавливает nginx с basic auth для доступа к temporal ui

### 1. Установить terraform

https://yandex.cloud/ru/docs/tutorials/infrastructure-management/terraform-quickstart

### 2. Установить переменные

Логин/пароль для basic auth

В файл infra/terraform/terraform.tfvars
```
nginx_username      = "{username}"
nginx_password      = "{something-strong}"
ssh_public_key_path = "~/.ssh/smarthome.pub"
```

### 3. Поднятие VM

```bash
cd infra/terraform
terraform init
terraform apply
```

### 4. Поднятие пайплайна построения каталога

```bash
ssh ubuntu@<ip>
sudo git clone https://github.com/Intelligent-Smart-Home-Design-System/monorepo /opt/pipeline
sudo chown -R ubuntu:ubuntu /opt/pipeline

cat > /opt/pipeline/services/pipeline-worker/.env << 'EOF'
CATALOG_DATABASE_PASSWORD=something-strong
YANDEX_CLOUD_API_KEY=your-real-key
EOF

cd /opt/pipeline/services/pipeline-worker
make build
make up
```

### 5. access
http://{ip}:8080  - temporal ui
