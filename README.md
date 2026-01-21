# Монорепозиторий для разработки интеллектуального планировщика умного дома.

## Стек
Golang, ...

## Организация проекта
Создание нового сервиса — в папке services/ создаете папку с названием вашего сервиса.

Пример:  
├services/
├────simulation/  
├────────cmd/  
├────────internal/  
├──────── ...  
├────frontend/  
├──────── ...  
├docs/
.gitignore  
README.md

- Если вы сервис на go, то при создании папки сервиса сделайте: go mod init github.com/Intelligent-Smart-Home-Design-System/monorepo/services/your-service-name (новый сервис = новый модуль)
- Организация проекта на golang: [ссылка](https://github.com/golang-standards/project-layout/blob/master/README_ru.md)

## Ветки
- main: главная ветка (develop)
- feature/task-name (ветка с вашей задачей)

В main мержим готовые версии задач (с пул реквестом если требуется). Готовая версия == стабильный код, которым можете поделиться.

Ветка feature/task-name — ветка под разработку вашей фичи. Эту ветку отводите от main и делаете ваше задание, пушите промежуточные результаты. Когда задание готово — merge в main.
