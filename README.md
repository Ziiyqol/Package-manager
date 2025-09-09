# Package-Manager 
## Данные сервис выполняет следующие функции: 
- Упаковывает файлы в архив, заливает их на сервер по SSH
- Скачивает файлы архивов по SSH и распаковывает
- Работает с файлами .json и .yaml 

## Переменные окружения сервиса:
- PM_SSH_USER
- PM_SSH_HOST
- PM_SSH_PORT (по умолчанию 22)
- PM_SSH_KEY

### Пример файла пакета для упаковки: 

```
packet.json
{
 "name": "packet-1",
 "ver": "1.10",
 "targets": [
  "./archive_this1/*.txt",
  {"path", "./archive_this2/*", "exclude": "*.tmp"},
 ]
 packets: {
  {"name": "packet-3", "ver": "<="2.0" },
 }
}
```
### Пример файла для распаковки:

```
packages.json
{
 "packages": [
  {"name": "packet-1", "ver": ">=1.10"},
  {"name": "packet-2" },
  {"name": "packet-3", "ver": "<="1.10" },
 ]
}
```


## Commandline tools с командами:

- pm create ./packet.json
- pm update ./packages.json

