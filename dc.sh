#!/usr/bin/env bash

# 创建子网
docker network create --subnet 172.20.0.0/16 --gateway 172.20.0.1 zk_kafka
# 执行docker-compose命令进行搭建
docker-compose -f dc.yml up -d
# 停止容器，并且删除实例
docker-compose -f dc.yml rm -fsv
# 输入 docker ps -a 命令如能查看到我们启动的三个服务且处于运行状态说明部署成功
# 测试kafka
# 输入 docker exec -it kafka0 bash 进入kafka0容器，并执行如下命令创建topic
# cd /opt/kafka_2.13-2.6.0/bin/
# ./kafka-topics.sh --create --topic chat --partitions 5 --zookeeper zookeeper:2181 --replication-factor 3
# 输入如下命令开启生产者
# ./kafka-console-producer.sh --broker-list kafka0:9092 --topic chat
# 开启另一个shell界面进入 kafka2 容器并执行下列命令开启消费者
# ./kafka-console-consumer.sh --bootstrap-server kafka2:9094 --topic chat --from-beginning


# 创建子网时出现如下错误
# docker network create --subnet 172.20.0.0/16 --gateway 172.20.0.1 zk_kafka
# Error response from daemon: Pool overlaps with other one on this address space
# networks参数下手动指定了subnet地址，此地址发生了冲突。
# docker network ls # 查看docker网卡
# docker network inspect <网卡id> # 查看具体信息，找到与subnet冲突的是哪个
# docker network rm <网卡id> # 删除冲突的网卡

# 本机连接时，在/etc/hosts中添加以下域名映射
# 127.0.0.1       kafka0
# 127.0.0.1       kafka1
# 127.0.0.1       kafka2


# zkui 参见 https://github.com/juris/docker-zkui/blob/master/Dockerfile
# http://127.0.0.1:9090/ admin/manager

# kafka-manager
# http://127.0.0.1:9000/ admin/admin

