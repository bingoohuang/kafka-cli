# kafka-cli

cli tools for kafka in golang

## docker-compose

1. [Setting up a simple Kafka cluster with docker for testing](http://www.smartjava.org/content/setting-up-kafka-cluster-docker-copy/)
1. [3 broker Kafka cluster and 3 node zk within docker-compose.](https://github.com/zoidbergwill/docker-compose-kafka/blob/master/docker-compose.yml)
1. `docker-compose up`、`docker-compose rm -fsv`

## Cluster Manager for Apache Kafka

1. TO CHECK [docker部署kafka集群](https://www.cnblogs.com/zisefeizhu/p/14151317.html)
1. TO CHECK [Docker Compose安装kafka和springboot整合kafka](https://www.cnblogs.com/Lambquan/p/13649715.html)
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

## kafka 版本

Version | Released | Major Changes
--- | ---| ---
2.7.0|2020-12-21|TCP Timeout ...
2.6.1|2021-01-07|...
2.6.0|2020-08-03|TLSv1.3, Streams, ZK3.5.8, ...
2.5.1|2020-08-10|...
2.5.0|2020-04-15|Incremental rebalance for consumer, Scala 2.12+
2.4.1|2020-03-12|...
2.4.0|2019-12-16|MirrorMaker,New Java authorizer,Admin API for replica reassignment...
2.3.1|2019-10-23|...
2.3.0|2019-06-25|incremental AlterConfigs API...
2.2.2|2019-12-01|...
2.2.1|2019-06-01|...
2.2.0|2019-03-22|SASL/SSL,API improvement...
2.1.1|2019-02-15|...
2.1.0|2018-11-20|Java11, zlib,
2.0.1|2018-11-09|...
2.0.0|2018-07-30|ACLs,  Kafka Streams improvement...
1.1.1|2018-07-19|...
1.1.0|2018-03-28|Controller improvements, Improvements Streams API...
1.0.2|2018-07-08|...
1.0.1|2018-03-05|...
1.0.0|2017-11-01|...
0.11.0.3|2018-07-02|...
0.11.0.2|2017-12-17|...
0.11.0.1|2017-09-13|...
0.11.0.0|2017-06-28|...
0.10.2.2|2018-07-02|...
0.10.2.1|2017-04-26|...
0.10.2.0|2017-02-21|...
0.10.1.1|2016-12-20|...
0.10.1.0|2016-10-20|...
0.10.0.1|2016-08-10|...
0.10.0.0|2016-05-22|...
0.9.0.1|2016-02-19|...
0.9.0.0|2015-11-23|...
0.8.2.1|2015-03-11|...
0.8.2.0|2015-02-02|...

### Kafka概述

Kafka是目前主流的分布式消息引擎及流处理平台，经常用做企业的消息总线、实时数据管道，有的还把它当做存储系统来使用。早期Kafka的定位是一个高吞吐的分布式消息系统，但随着版本的不断迭代，目前已经发展成为一个分布式流处理平台了。

Kafka遵循生产者消费者模式，生产者发送消息到Broker中某一个Topic的具体分区里，消费者从一个或多个分区中拉取数据进行消费。目前Kafka主要提供了四套核心的API，Producer API、Consumer API、Streams API与Connector API。随着Kafka不同版本的发布，API的支持也有所不同，具体也可参考下面的介绍。

### Kafka版本规则

在Kafka 1.0.0之前基本遵循4位版本号，比如Kafka 0.8.2.0、Kafka 0.11.0.3等。而从1.0.0开始Kafka就告别了4位版本号，遵循 Major.Minor.Patch 的版本规则，其中Major表示大版本，通常是一些重大改变，因此彼此之间功能可能会不兼容；Minor表示小版本，通常是一些新功能的增加；最后Patch表示修订版，主要为修复一些重点Bug而发布的版本。比如Kafka 2.1.1，大版本就是2，小版本是1，Patch版本为1，是为修复Bug发布的第1个版本。

### Kafka版本演进

Kafka总共发布了7个大版本，分别是0.7.x、0.8.x、0.9.x、0.10.x、0.11.x、1.x及2.x版本。截止目前，最新版本是Kafka 2.4.0，也是最新稳定版本。

### 0.7.x版本

这是很老的Kafka版本，它只有基本的消息队列功能，连消息副本机制都没有，不建议使用。

### 0.8.x版本

两个重要特性，一个是Kafka 0.8.0增加了副本机制，另一个是Kafka 0.8.2.0引入了新版本Producer API。新旧版本Producer API如下：

```java
// 旧版本Producer
kafka.javaapi.producer.Producer<K,V>

// 新版本Producer
org.apache.kafka.clients.producer.KafkaProducer<K,V>
```

与旧版本相比，新版本Producer API有点不同:

1. 连接Kafka方式上，旧版本的生产者及消费者API连接的是Zookeeper，而新版本则连接的是Broker；
1. 新版Producer采用异步方式发送消息，比之前同步发送消息的性能有所提升。但此时的新版Producer API尚不稳定，不建议生产使用。

### 0.9.x版本

Kafka 0.9 是一个重大的版本迭代，增加了非常多的新特性，主要体现在三个方面：

1. 安全方面：在0.9.0之前，Kafka安全方面的考虑几乎为0。Kafka 0.9.0 在安全认证、授权管理、数据加密等方面都得到了支持，包括支持Kerberos等。
1. 新版本Consumer API：Kafka 0.9.0 重写并提供了新版消费端API，使用方式也是从连接Zookeeper切到了连接Broker，但是此时新版Consumer API也不太稳定、存在不少Bug，生产使用可能会比较痛苦；而0.9.0版本的Producer API已经比较稳定了，生产使用问题不大。
1. Kafka Connect：Kafka 0.9.0 引入了新的组件 Kafka Connect ，用于实现Kafka与其他外部系统之间的数据抽取。

### 0.10.x版本

Kafka 0.10 是一个重要的大版本，因为Kafka 0.10.0.0 引入了 Kafka Streams，使得Kafka不再仅是一个消息引擎，而是往一个分布式流处理平台方向发展。

0.10 大版本包含两个小版本：0.10.1 和 0.10.2，它们的主要功能变更都是在 Kafka Streams 组件上。

值得一提的是，自 0.10.2.2 版本起，新版本 Consumer API 已经比较稳定了，而且 Producer API 的性能也得到了提升，因此对于使用 0.10.x 大版本的用户，建议使用或升级到 Kafka 0.10.2.2 版本。

### 0.11.x版本

Kafka 0.11 是一个里程碑式的大版本，主要有两个大的变更:

1. Kafka从这个版本开始支持 Exactly-Once 语义即精准一次语义，主要是实现了Producer端的消息幂等性，以及事务特性，这对于Kafka流式处理具有非常大的意义。
2. 另一个重大变更是Kafka消息格式的重构，Kafka 0.11主要为了实现Producer幂等性与事务特性，重构了投递消息的数据结构。这一点非常值得关注，因为Kafka 0.11之后的消息格式发生了变化，所以我们要特别注意Kafka不同版本间消息格式不兼容的问题。

### 1.x版本

Kafka 1.x 更多的是Kafka Streams方面的改进，以及Kafka Connect的改进与功能完善等。但仍有两个重要特性:

1. Kafka 1.0.0实现了磁盘的故障转移，当Broker的某一块磁盘损坏时数据会自动转移到其他正常的磁盘上，Broker还会正常工作，这在之前版本中则会直接导致Broker宕机，因此Kafka的可用性与可靠性得到了提升；
2. Kafka 1.1.0开始支持副本跨路径迁移，分区副本可以在同一Broker不同磁盘目录间进行移动，这对于磁盘的负载均衡非常有意义。

### 2.x版本：

Kafka 2.x 更多的也是Kafka Streams、Connect方面的性能提升与功能完善，以及安全方面的增强等。一个使用特性，Kafka 2.1.0开始支持ZStandard的压缩方式，提升了消息的压缩比，显著减少了磁盘空间与网络io消耗。

### Kafka版本建议

1. 遵循一个基本原则，Kafka客户端版本和服务端版本应该保持一致，否则可能会遇到一些问题。
1. 根据是否用到了Kafka的一些新特性来选择，假如要用到Kafka生产端的消息幂等性，那么建议选择Kafka 0.11 或之后的版本。
1. 选择一个自己熟悉且稳定的版本，如果说没有比较熟悉的版本，建议选择一个较新且稳定、使用比较广泛的版本。

### 主要参考：

1. [Apache Kafka 版本演进及特性介绍](https://cloud.tencent.com/developer/article/1596747)
1. [Kafka huxihx blog](https://www.cnblogs.com/huxi2b/)
2. [Kafka downloads with release notes](http://kafka.apache.org/downloads)
