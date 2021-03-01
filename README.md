# kafka-cli

cli tools for kafka in golang

## docker-compose

1. [Setting up a simple Kafka cluster with docker for testing](http://www.smartjava.org/content/setting-up-kafka-cluster-docker-copy/)
1. [3 broker Kafka cluster and 3 node zk within docker-compose.](https://github.com/zoidbergwill/docker-compose-kafka/blob/master/docker-compose.yml)
1. `docker-compose up`、`docker-compose rm -fsv`

## Cluster Manager for Apache Kafka

1. [obsidiandynamics/kafdrop](https://github.com/obsidiandynamics/kafdrop)
1. [yahoo/CMAK Kafka Manager](https://github.com/yahoo/CMAK)
1. [didi/Logi-KafkaManager](https://github.com/didi/Logi-KafkaManager)
1. [kafka-tools ConfluentInc slides](https://www.slideshare.net/ConfluentInc/show-me-kafka-tools-that-will-increase-my-productivity-stephane-maarek-datacumulus-kafka-summit-london-2019)

```sh
$ kafka-cli console-consumer --brokers 192.168.217.14:9092                               
2021/03/01 18:08:32 Config:{"Brokers":"192.168.217.14:9092","Version":"0.10.0.0","Topic":"kafka-cli.topic","Partitions":"all","Offset":"newest","BufferSize":256}
Partition: 0 Offset: 13 Key:  Value: [bingoohuang]
Partition: 0 Offset: 14 Key:  Value: [bingoohuangaaa]
```

```sh
$ kafka-cli console-producer --brokers  192.168.217.14:9092 --value bingoohuang           
2021/03/01 18:08:45 Config:{"Brokers":"192.168.217.14:9092","Version":"0.10.0.0","Topic":"kafka-cli.topic","Value":"bingoohuang","Partition":-1}
topic=kafka-cli.topic   partition=0     offset=13
$ kafka-cli console-producer --brokers  192.168.217.14:9092 --value bingoohuangaaa 
2021/03/01 18:08:49 Config:{"Brokers":"192.168.217.14:9092","Version":"0.10.0.0","Topic":"kafka-cli.topic","Value":"bingoohuangaaa","Partition":-1}
topic=kafka-cli.topic   partition=0     offset=14
```
