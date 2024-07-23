#!/bin/sh

# Устанавливаем переменную окружения KAFKA_CLUSTER_ID
export KAFKA_CLUSTER_ID=e2e2f794-534d-45a9-ad64-b98a7c09d55d

# Проверяем, что переменная окружения установлена
echo "KAFKA_CLUSTER_ID=${KAFKA_CLUSTER_ID}"

# Выполняем команду для проверки KAFKA_CLUSTER_ID
/usr/local/bin/dub ensure KAFKA_CLUSTER_ID=${KAFKA_CLUSTER_ID}

# Запускаем Kafka
exec /etc/confluent/docker/run
