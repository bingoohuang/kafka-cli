package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Shopify/sarama"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type ConsumerCmd struct {
	KakfaConnect

	Group    string `json:",omitempty"`
	Topics   string `json:",omitempty"`
	Assignor string `json:",omitempty"`
	Oldest   bool   `json:",omitempty"`
}

func initConsumerCmd(root *cobra.Command) {
	c := ConsumerCmd{}
	cmd := &cobra.Command{
		Use:   "consumer",
		Short: "Starts consuming the given Kafka topics and logs the consumed messages.",
		Long:  `eg: kafka-cli consume -brokers="127.0.0.1:9092" -topics="sarama" -group="example"`,
		Run:   func(cmd *cobra.Command, args []string) { c.run() },
	}
	root.AddCommand(cmd)

	f := cmd.Flags()
	c.KakfaConnect.InitFlags(f)

	f.StringVar(&c.Group, "group", "kafka-cli.group", "Kafka consumer group definition")
	f.StringVar(&c.Topics, "topics", "kafka-cli.topic", "Kafka topics to be consumed, as a comma separated list")
	f.StringVar(&c.Assignor, "assignor", "range", "Consumer group partition assignment strategy (range, rr/roundrobin, sticky)")
	f.BoolVar(&c.Oldest, "oldest", true, "Kafka consumer consume initial offset from oldest")
}

func (r *ConsumerCmd) run() {
	cmdJSON, _ := json.Marshal(r)
	log.Printf("Starting consumer with config: %s", cmdJSON)

	if r.Verbose {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}

	/**
	 * Construct a new Sarama configuration.
	 * The Kafka cluster version has to be defined before the consumer/producer is initialized.
	 */
	cnf := sarama.NewConfig()
	r.SetupVersion(cnf)
	r.SetupTlSConfig(cnf)

	switch r.Assignor {
	case "sticky":
		cnf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	case "roundrobin", "rr":
		cnf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	case "range":
		cnf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	default:
		log.Panicf("Unrecognized consumer group partition assignor: %s", r.Assignor)
	}

	if r.Oldest {
		cnf.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	/**
	 * Setup a new Sarama consumer group
	 */
	consumer := Consumer{
		ready: make(chan bool),
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup(strings.Split(r.Brokers, ","), r.Group, cnf)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, strings.Split(r.Topics, ","), &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Println("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Println("terminating: context cancelled")
	case <-sigterm:
		log.Println("terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Panicf("Error closing client: %v", err)
	}
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (r *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the r as ready
	close(r.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (r *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

const layout = "2006-01-02 15:03:04.000"

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (r *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for m := range claim.Messages() {
		headers := ""
		for _, h := range m.Headers {
			if headers == "" {
				headers = " Headers "
			} else {
				headers += ", "
			}
			headers += string(h.Key) + ": " + string(h.Value)
		}
		log.Printf("Msg claimed at %s/%s topic: %s, offset:%d, partition:%d,%s K:%s, V: %s",
			m.Timestamp.Format(layout), m.BlockTimestamp.Format(layout),
			m.Topic, m.Offset, m.Partition, headers, m.Key, pretty.Color(m.Value, nil))
		session.MarkMessage(m, "")
	}

	return nil
}
