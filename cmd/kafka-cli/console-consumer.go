package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/Shopify/sarama"
	"github.com/spf13/cobra"
)

type ConsoleConsumerCmd struct {
	KakfaConnect

	Topic      string `json:",omitempty"`
	Partitions string `json:",omitempty"`
	Oldest     bool   `json:",omitempty"`

	BufferSize int `json:",omitempty"`
}

func initConsoleConsumerCmd(root *cobra.Command) {
	c := ConsoleConsumerCmd{}
	cmd := &cobra.Command{
		Use:   "console-consumer",
		Short: "consume partitions of a topic and print the messages on the standard output.",
		Long: `# Minimum invocation
kafka-cli console-consumer -topic=test -brokers=kafka1:9092

# It will pick up a KAFKA_PEERS environment variable
export KAFKA_PEERS=kafka1:9092,kafka2:9092,kafka3:9092
kafka-cli console-consumer -topic=test

# You can specify the offset you want to start at. It can be either
# oldest, newest. The default is newest.
kafka-cli console-consumer -topic=test -offset=oldest
kafka-cli console-consumer -topic=test -offset=newest

# You can specify the partition(s) you want to consume as a comma-separated
# list. The default is all.
kafka-cli console-consumer -topic=test -partitions=1,2,3

# Display all command line options
kafka-cli console-consumer -help`,
		Run: func(cmd *cobra.Command, args []string) { c.run() },
	}
	root.AddCommand(cmd)

	f := cmd.Flags()
	c.KakfaConnect.InitFlags(f)
	f.StringVar(&c.Topic, "topic", "kafka-cli.topic", "Topic to consume")
	f.StringVar(&c.Partitions, "partitions", "all", "Partitions to consume, can be 'all' or comma-separated numbers")
	f.BoolVar(&c.Oldest, "oldest", true, "Consume initial offset from oldest")
	f.IntVar(&c.BufferSize, "buffer-size", 256, "Buffer size of the message channel.")
}

func (r *ConsoleConsumerCmd) run() {
	cmdJSON, _ := json.Marshal(r)
	log.Printf("Config:%s", cmdJSON)

	if r.Verbose {
		sarama.Logger = log.Default()
	}

	var initialOffset = sarama.OffsetNewest
	if r.Oldest {
		initialOffset = sarama.OffsetOldest
	}

	cnf := sarama.NewConfig()
	r.SetupVersion(cnf)
	r.SetupTlSConfig(cnf)

	c, err := sarama.NewConsumer(strings.Split(r.Brokers, ","), cnf)
	if err != nil {
		printErrorAndExit(69, "Failed to start consumer: %s", err)
	}

	partitionList, err := r.getPartitions(c)
	if err != nil {
		printErrorAndExit(69, "Failed to get the list of partitions: %s", err)
	}

	var (
		messages = make(chan *sarama.ConsumerMessage, r.BufferSize)
		closing  = make(chan struct{})
		wg       sync.WaitGroup
	)

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
		<-signals
		log.Println("Initiating shutdown of consumer...")
		close(closing)
	}()

	for _, partition := range partitionList {
		pc, err := c.ConsumePartition(r.Topic, partition, initialOffset)
		if err != nil {
			printErrorAndExit(69, "Failed to start consumer for partition %d: %s", partition, err)
		}

		go func(pc sarama.PartitionConsumer) {
			<-closing
			pc.AsyncClose()
		}(pc)

		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for message := range pc.Messages() {
				messages <- message
			}
		}(pc)
	}

	go func() {
		for msg := range messages {
			fmt.Printf("Partition: %d", msg.Partition)
			fmt.Printf(" Offset: %d", msg.Offset)
			fmt.Printf(" Key: %s", string(msg.Key))
			fmt.Printf(" Value: [%s]\n", string(msg.Value))
		}
	}()

	wg.Wait()
	log.Println("Done consuming topic", r.Topic)
	close(messages)

	if err := c.Close(); err != nil {
		log.Println("Failed to close consumer: ", err)
	}
}

func (r *ConsoleConsumerCmd) getPartitions(c sarama.Consumer) ([]int32, error) {
	if r.Partitions == "all" {
		return c.Partitions(r.Topic)
	}

	tmp := strings.Split(r.Partitions, ",")
	var pList []int32
	for i := range tmp {
		val, err := strconv.ParseInt(tmp[i], 10, 32)
		if err != nil {
			return nil, err
		}
		pList = append(pList, int32(val))
	}

	return pList, nil
}

func printErrorAndExit(code int, format string, values ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
	_, _ = fmt.Fprintln(os.Stderr)
	os.Exit(code)
}

func printUsageErrorAndExit(format string, values ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", fmt.Sprintf(format, values...))
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, "Available command line options:")
	flag.PrintDefaults()
	os.Exit(64)
}
