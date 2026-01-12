# Монорепозиторий для разработки интеллектуального планировщика умного дома.

## Стек
Golang, ...

## Организация проекта
Создание нового сервиса — в корне монорепо папка с названием сервиса.

Пример:  
simulation/  
├──cmd/  
├──internal/  
├── ...  
frontend/  
├── ...  
.gitignore  
README.md

- Если вы сервис на go, то при создании папки сервиса сделайте: go mod init github.com/Intelligent-Smart-Home-Design-System/monorepo/your-service-name (новый сервис = новый модуль)
- Организация проекта на golang: [ссылка](https://github.com/golang-standards/project-layout/blob/master/README_ru.md)

## Ветки
- main: главная ветка (прод)
- feature/task-name (ветка с вашей задачей)

Ветка main — продакшн. В неё мержим готовые фичи (с пул реквестом если требуется).

Ветка feature/task-name — ветка под разработку вашей фичи. Эту ветку отводите от main и делаете ваше задание. Когда задание готово — merge в main.
