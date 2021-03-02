package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/gohxs/readline"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type ConsoleProducerCmd struct {
	KakfaConnect

	Topic       string `json:",omitempty"`
	Headers     string `json:",omitempty"`
	Key         string `json:",omitempty"`
	Value       string `json:",omitempty"`
	Partitioner string `json:",omitempty"`
	Partition   int    `json:",omitempty"`
	ShowMetrics bool   `json:",omitempty"`
	Silent      bool   `json:",omitempty"`
}

func initConsoleProducerCmd(root *cobra.Command) {
	c := ConsoleProducerCmd{}
	cmd := &cobra.Command{
		Use:   "console-producer",
		Short: "produce a single message to Kafka.",
		Long: `# Minimum invocation
kafka-cli console-producer -topic=test -value=value -brokers=kafka1:9092

# It will pick up a KAFKA_PEERS environment variable
export KAFKA_PEERS=kafka1:9092,kafka2:9092,kafka3:9092
kafka-cli console-producer -topic=test -value=value

# It will read the value from stdin by using pipes
echo "hello world" | kafka-cli console-producer -topic=test

# Specify a key:
echo "hello world" | kafka-cli console-producer -topic=test -key=key

# Partitioning: by default, kafka-console-producer will partition as follows:
# - manual partitioning if a -partition is provided
# - hash partitioning by key if a -key is provided
# - random partitioning otherwise.
#
# You can override this using the -partitioner argument:
echo "hello world" | kafka-console-producer -topic=test -key=key -partitioner=random

# Display all command line options
kafka-cli console-producer -help`,
		Run: func(cmd *cobra.Command, args []string) { c.run() },
	}
	root.AddCommand(cmd)

	f := cmd.Flags()
	c.KakfaConnect.InitFlags(f)
	f.StringVar(&c.Headers, "headers", "", "The headers of the message to produce. Example: -headers=foo:bar,bar:foo")
	f.StringVar(&c.Topic, "topic", "kafka-cli.topic", "REQUIRED: the topic to produce to")
	f.StringVar(&c.Key, "key", "", "Message key to produce. Can be empty.")
	f.StringVar(&c.Value, "value", "", "Message value to produce. You can also provide the value on stdin or type in later in prompt.")
	f.StringVar(&c.Partitioner, "partitioner", "", "The partitioning scheme to use. hash/manual/random")
	f.IntVar(&c.Partition, "partition", -1, "The partition to produce to.")
	f.BoolVar(&c.ShowMetrics, "metrics", false, "Output metrics on successful publish to stderr")
	f.BoolVar(&c.Silent, "silent", false, "Turn off printing the message's topic, partition, and offset to stdout")
}

func (r *ConsoleProducerCmd) run() {
	cmdJSON, _ := json.Marshal(r)
	log.Printf("Config:%s", cmdJSON)

	if r.Brokers == "" {
		printUsageErrorAndExit("no -brokers specified. Alternatively, set the KAFKA_PEERS environment variable")
	}

	if r.Topic == "" {
		printUsageErrorAndExit("no -topic specified")
	}

	if r.Verbose {
		sarama.Logger = log.Default()
	}

	cnf := sarama.NewConfig()
	cnf.Producer.RequiredAcks = sarama.WaitForAll
	cnf.Producer.Return.Successes = true

	r.SetupVersion(cnf)
	r.SetupTlSConfig(cnf)

	switch r.Partitioner {
	case "":
		if r.Partition >= 0 {
			cnf.Producer.Partitioner = sarama.NewManualPartitioner
		} else {
			cnf.Producer.Partitioner = sarama.NewHashPartitioner
		}
	case "hash":
		cnf.Producer.Partitioner = sarama.NewHashPartitioner
	case "random":
		cnf.Producer.Partitioner = sarama.NewRandomPartitioner
	case "manual":
		cnf.Producer.Partitioner = sarama.NewManualPartitioner
		if r.Partition == -1 {
			printUsageErrorAndExit("-partition is required when partitioning manually")
		}
	default:
		printUsageErrorAndExit(fmt.Sprintf("Partitioner %s not supported.", r.Partitioner))
	}

	msg := &sarama.ProducerMessage{Topic: r.Topic, Partition: int32(r.Partition)}
	if r.Key != "" {
		msg.Key = sarama.StringEncoder(r.Key)
	}

	var readlineUI bool

	if r.Value != "" {
		msg.Value = sarama.StringEncoder(r.Value)
	} else if stdinAvailable() {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			printErrorAndExit(66, "Failed to read data from the standard input: %s", err)
		}

		msg.Value = sarama.ByteEncoder(bytes.TrimSpace(data))
	} else {
		readlineUI = true
	}

	r.appendHeaders(msg)

	p, err := sarama.NewSyncProducer(strings.Split(r.Brokers, ","), cnf)
	if err != nil {
		printErrorAndExit(69, "Failed to open Kafka p: %s", err)
	}
	defer func() {
		if err := p.Close(); err != nil {
			log.Println("Failed to close Kafka p cleanly:", err)
		}
	}()

	if !readlineUI {
		r.send(p, msg, cnf)
		return
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 "> ",
		HistoryFile:            "/tmp/kafka-cli",
		DisableAutoSaveHistory: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()
	var lastErrInterrupt time.Time

	for {
		l, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if time.Since(lastErrInterrupt) < 3*time.Second {
				os.Exit(0)
			}

			lastErrInterrupt = time.Now()
			continue
		}

		if err != nil {
			break
		}

		lastErrInterrupt = time.Time{}

		l = strings.TrimSpace(l)
		if len(l) == 0 {
			continue
		}
		rl.SaveHistory(l)

		msg.Value = sarama.StringEncoder(l)
		r.send(p, msg, cnf)
	}

}

func (r *ConsoleProducerCmd) send(p sarama.SyncProducer, msg *sarama.ProducerMessage, cnf *sarama.Config) {
	partition, offset, err := p.SendMessage(msg)
	if err != nil {
		printErrorAndExit(69, "Failed to produce msg: %s", err)
	} else if !r.Silent {
		fmt.Printf("topic=%s\tpartition=%d\toffset=%d\n", r.Topic, partition, offset)
	}
	if r.ShowMetrics {
		metrics.WriteOnce(cnf.MetricRegistry, os.Stderr)
	}
}

func (r *ConsoleProducerCmd) appendHeaders(msg *sarama.ProducerMessage) {
	if r.Headers == "" {
		return
	}

	var hdrs []sarama.RecordHeader
	for _, h := range strings.Split(r.Headers, ",") {
		if header := strings.Split(h, ":"); len(header) != 2 {
			printUsageErrorAndExit("-header should be key:value. Example: -headers=foo:bar,bar:foo")
		} else {
			hdrs = append(hdrs, sarama.RecordHeader{
				Key:   []byte(header[0]),
				Value: []byte(header[1]),
			})
		}
	}

	msg.Headers = hdrs
}

func stdinAvailable() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
