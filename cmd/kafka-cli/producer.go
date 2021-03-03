package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/gohxs/readline"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/cobra"
)

type ProducerCmd struct {
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

func initProducerCmd(root *cobra.Command) {
	c := ProducerCmd{}
	cmd := &cobra.Command{
		Use:   "producer",
		Short: "produce a single message to Kafka.",
		Long: `# Minimum invocation
kafka-cli producer -topic=test -value=value -brokers=kafka1:9092

# It will pick up a KAFKA_PEERS environment variable
export KAFKA_PEERS=kafka1:9092,kafka2:9092,kafka3:9092
kafka-cli producer -topic=test -value=value

# It will read the value from stdin by using pipes
echo "hello world" | kafka-cli producer -topic=test

# Specify a key:
echo "hello world" | kafka-cli producer -topic=test -key=key

# Partitioning: by default, kafka-console-producer will partition as follows:
# - manual partitioning if a -partition is provided
# - hash partitioning by key if a -key is provided
# - random partitioning otherwise.
#
# You can override this using the -partitioner argument:
echo "hello world" | kafka-console-producer -topic=test -key=key -partitioner=random

# Display all command line options
kafka-cli producer -help`,
		Run: func(cmd *cobra.Command, args []string) { c.run() },
	}
	root.AddCommand(cmd)

	f := cmd.Flags()
	c.KakfaConnect.InitFlags(f)
	f.StringVar(&c.Headers, "headers", "", "The headers of the message to produce. Example: -headers=foo:bar,bar:foo")
	f.StringVar(&c.Topic, "topic", "kafka-cli.topic", "Topic to produce to")
	f.StringVar(&c.Key, "key", "", "Message key to produce. Can be empty.")
	f.StringVar(&c.Value, "value", "", "Message value to produce. Or on stdin or type in later in prompt.")
	f.StringVar(&c.Partitioner, "partitioner", "", "The partitioning scheme to use. hash/manual/random")
	f.IntVar(&c.Partition, "partition", -1, "The partition to produce to.")
	f.BoolVar(&c.ShowMetrics, "metrics", false, "Output metrics on successful publish to stderr")
	f.BoolVar(&c.Silent, "silent", false, "Turn off printing the message's topic, partition, and offset to stdout")
}

func (r *ProducerCmd) run() {
	cmdJSON, _ := json.Marshal(r)
	log.Printf("Starting producer with config: %s", cmdJSON)

	if r.Verbose {
		sarama.Logger = log.Default()
	}

	cnf := sarama.NewConfig()
	cnf.Producer.RequiredAcks = sarama.WaitForAll
	cnf.Producer.Return.Successes = true

	r.SetupVersion(cnf)
	r.SetupTlSConfig(cnf)
	r.SetupPartitioner(cnf)

	msg := &sarama.ProducerMessage{Topic: r.Topic, Partition: int32(r.Partition)}
	msg.Key = sarama.StringEncoder(r.Key)
	readlineUI := r.SetupValue(msg)

	p, err := sarama.NewSyncProducer(strings.Split(r.Brokers, ","), cnf)
	if err != nil {
		printErrorAndExit(69, "Failed to open Kafka p: %s", err)
	}
	defer Close(p)

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

	var lastInterrupt time.Time

	defaultHeaders := ParseHeaders(r.Headers)

	for {
		l, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if time.Since(lastInterrupt) < 3*time.Second {
				os.Exit(0)
			}

			lastInterrupt = time.Now()
			continue
		}

		if err != nil {
			break
		}

		// Reset
		lastInterrupt = time.Time{}

		l = strings.TrimSpace(l)
		if len(l) == 0 {
			continue
		}
		rl.SaveHistory(l)

		ParseKafkaMessageJSON(msg, l, defaultHeaders)
		r.send(p, msg, cnf)
	}
}

func Close(p io.Closer) {
	if err := p.Close(); err != nil {
		log.Println("Failed to close Kafka p cleanly:", err)
	}
}

func (r *ProducerCmd) SetupValue(msg *sarama.ProducerMessage) (readlineUI bool) {
	if r.Value != "" {
		msg.Value = sarama.StringEncoder(r.Value)
		return false
	}

	if stdinAvailable() {
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			printErrorAndExit(66, "Failed to read data from the standard input: %s", err)
		}

		msg.Value = sarama.ByteEncoder(bytes.TrimSpace(data))
		return false
	}

	return true
}

func (r *ProducerCmd) SetupPartitioner(cnf *sarama.Config) {
	cnf.Producer.Partitioner = ParsePartitioner(r.Partitioner, r.Partition)
}

func ParsePartitioner(partitioner string, partition int) sarama.PartitionerConstructor {
	switch partitioner {
	case "":
		if partition >= 0 {
			return sarama.NewManualPartitioner
		}
		return sarama.NewHashPartitioner
	case "hash":
		return sarama.NewHashPartitioner
	case "random":
		return sarama.NewRandomPartitioner
	case "manual":
		if partition == -1 {
			printUsageErrorAndExit("-partition is required when partitioning manually")
		}
		return sarama.NewManualPartitioner
	default:
		printUsageErrorAndExit(fmt.Sprintf("Partitioner %s not supported.", partitioner))
		return nil
	}
}

func (r *ProducerCmd) send(p sarama.SyncProducer, msg *sarama.ProducerMessage, cnf *sarama.Config) {
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

type KafkaMessage struct {
	K string            `json:"k,omitempty"`
	V json.RawMessage   `json:"v,omitempty"`
	H map[string]string `json:"h,omitempty"`
}

func ParseKafkaMessageJSON(msg *sarama.ProducerMessage, s string, defaultHeaders []sarama.RecordHeader) {
	v := &KafkaMessage{}
	if err := json.Unmarshal([]byte(s), v); err != nil {
		msg.Value = sarama.StringEncoder(s)
		return
	}

	msg.Key = sarama.StringEncoder(v.K)
	msg.Value = sarama.ByteEncoder(v.V)
	var hdrs []sarama.RecordHeader
	for k, v := range v.H {
		hdrs = append(hdrs, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}
	if len(hdrs) > 0 {
		msg.Headers = hdrs
	} else {
		msg.Headers = defaultHeaders
	}
}

func ParseHeaders(headers string) []sarama.RecordHeader {
	var hdrs []sarama.RecordHeader

	if headers == "" {
		return hdrs
	}

	for _, h := range strings.Split(headers, ",") {
		if header := strings.Split(h, ":"); len(header) != 2 {
			printUsageErrorAndExit("--header should be key:value. Example: -headers=foo:bar,bar:foo")
		} else {
			hdrs = append(hdrs, sarama.RecordHeader{
				Key:   []byte(header[0]),
				Value: []byte(header[1]),
			})
		}
	}

	return hdrs
}

func stdinAvailable() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
