---
layout: post
title: go操作kafka
date: 2022-07-24
tags:  MQ
---

# Kafka简介
- 消息队列，提供了消息持久化的功能，高效的IO

核心原理：将消息以**Append Log**的方式顺序存储在磁盘上

- 核心概念
  - Producer 生产者即向kafka发送消息的一方  可以有多个
  - Consumer 消费者即从kafka接收消息的一方  可以有多个
  - Broker 即kafka集群中的服务器
  - Topic  主题即kafka会按照topic来分发消息
  - Partition kafka中的topic可以包含多个partition，partition**内部消息有序**，partition**之间的消息不保证顺序**

![](/images/posts/img/Snipaste_2022-07-24_09-46-06.png)



# 环境部署
使用Docker-Compose来部署 https://developer.confluent.io/quickstart/kafka-docker/

kafka内部使用到了Zookeeper，使用Zookeeper来维护集群，在集群内部进行选举，kafka broker需要借助Zookeeper来维护集群 




# golang代码实践
golang客户端
https://github.com/confluentinc/confluent-kafka-go

**生产者**
```
func main() {
    //实例化一个producer
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost"})
	if err != nil {
		panic(err)
	}

	defer p.Close()

	// 开一个协程去发送消息
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	// 异步的向topic中生成消息
	topic := "myTopic"
	for _, word := range []string{"Welcome", "to", "the", "Confluent", "Kafka", "Golang", "client"} {
		p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(word),
		}, nil)
	}

	// 在关闭之前会等待消息的传递
	p.Flush(15 * 1000)
}

```

**消费者**
```
func main() {

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost",
		"group.id":          "myGroup",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		panic(err)
	}

	c.SubscribeTopics([]string{"myTopic", "^aRegex.*[Tt]opic"}, nil)
    //死循环 一直消费消息
	for {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			fmt.Printf("Message on %s: %s\n", msg.TopicPartition, string(msg.Value))
		} else {
			fmt.Printf("Consumer error: %v (%v)\n", err, msg)
		}
	}

	c.Close()
}
```