# warpscout-mikrotik

Контейнер ARM64 для автоматического сканирования Cloudflare WARP с помощью
[niklzz/warpscout-tg](https://github.com/niklzz/warpscout-tg), генерации AWG `.conf`

## Настройки

Envs, list=`warpscout`:

| Key | Пример Value |
|---|---|
| SCAN_COMMAND | `scan -p awg -tg-only -exclude-node DME -conf warp.conf` |
| INTERVAL_SECONDS | `21600` |

Оба ключа обязательны. `21600` — пауза 6 часов после скана; `0` — один скан и выход.

Mounts, list=`warpscout`:

| src: каталог на роутере, пример | dst: фиксированный каталог контейнера |
|---|---|
| `/usb1/docker_configs/mihomo_mikrotik/awg` | `/output` |
| `/usb1/docker_configs/warpscout/state` | `/state` |

Можно заранее положить свои учётные данные в
`/state/warpscout-account.json`. Существующий файл используется без перерегистрации.
Отсутствующий файл создаётся через `warpscout register`. 

Конфиг публикуется только при успешном непустом результате.
После публикации записывается `/state/last-success`: UTC-время и уникальный токен.
Ошибка скана не меняет прежний конфиг или маркер. Маркер не содержит ключей.

## Сборка

На Ubuntu x86_64 с Podman:

```sh
sh build-podman.sh
```

С Docker Desktop:

```sh
docker buildx build --platform linux/arm64 --load -t warpscout-mikrotik:arm64 .
docker save --output warpscout-mikrotik-arm64.tar warpscout-mikrotik:arm64
```

По умолчанию сборка скачивает warpscout-tg из master. Для другого источника добавьте
`--build-arg WARPSCOUT_REPO=vernette/warpscout`. Для фиксации версии добавьте
`--build-arg WARPSCOUT_REF=<commit-sha>`. Фактический SHA записан в образе
в `/app/source-commit.txt`. Для обновления upstream используйте `--no-cache`,
чтобы повторно скачать исходники вместо использования кешированного слоя Git.

## MikroTik

Нужны RouterOS с поддержкой контейнеров, ARM64 и внешний накопитель.
Примеры подготовлены под RouterOS 7.24.5. Сканер должен иметь прямой выход в WAN;
не направляйте его трафик через уже работающий WARP.

Для существующего Bridge-Docker со шлюзом 192.168.254.1/24:

```routeros
/interface/veth/add name=WARPSCOUT address=192.168.254.31/24 gateway=192.168.254.1 dhcp=no
/interface/bridge/port/add bridge=Bridge-Docker interface=WARPSCOUT
/container/mounts/add list=warpscout src=/usb1/docker_configs/mihomo_mikrotik/awg dst=/output
/container/mounts/add list=warpscout src=/usb1/docker_configs/warpscout/state dst=/state
/container/envs/add list=warpscout key=SCAN_COMMAND value="scan -p awg -tg-only -exclude-node DME -conf warp_ru_5.conf"
/container/envs/add list=warpscout key=INTERVAL_SECONDS value="21600"
/container/add name=warpscout file=usb1/warpscout-mikrotik-arm64.tar interface=WARPSCOUT dns=1.1.1.1,8.8.8.8 envlists=warpscout mountlists=warpscout root-dir=/usb1/docker/warpscout logging=yes start-on-boot=no workdir=/state
```



## Лицензия и upstream

Обёртка и скрипты этого репозитория — MIT, см. LICENSE.
warpscout-tg — отдельный MIT-проект; его код скачивается при сборке.
Авторство и лицензия upstream: [niklzz/warpscout-tg](https://github.com/niklzz/warpscout-tg).
