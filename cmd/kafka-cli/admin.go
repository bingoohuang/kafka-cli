package main

import (
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/spf13/cobra"
	"strings"
)

type AdminCmd struct {
	KakfaConnect
	Query string
}

func initAdminCmd(root *cobra.Command) {
	c := AdminCmd{}
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "admin.",
		Long: `# List of topics
kafka-cli admin -brokers=kafka1:9092 -query=topics

# It will pick up a KAFKA_PEERS environment variable
export KAFKA_PEERS=kafka1:9092,kafka2:9092,kafka3:9092
kafka-cli admin -query=topics

# Display all command line options
kafka-cli admin -help`,
		Run: func(cmd *cobra.Command, args []string) { c.run() },
	}
	root.AddCommand(cmd)

	f := cmd.Flags()
	c.KakfaConnect.InitFlags(f)
	f.StringVar(&c.Query, "query", "", "query expression")
}

func (r *AdminCmd) run() {
	cnf := sarama.NewConfig()
	r.SetupVersion(cnf)
	r.SetupTlSConfig(cnf)

	cmd := queryCmdsRegistrar["topics"](r.Brokers, cnf)
	cmd.Execute()
}

type QueryCmd interface {
	Execute() error
}

var queryCmdsRegistrar = make(map[string]func(brokers string, config *sarama.Config) QueryCmd)

type BasicCmd struct {
	Brokers string
	Config  *sarama.Config
}

func init() {
	queryCmdsRegistrar["topics"] = func(brokers string, config *sarama.Config) QueryCmd {
		return &TopicsCmd{
			BasicCmd: BasicCmd{Brokers: brokers, Config: config},
		}
	}
}

type TopicsCmd struct {
	BasicCmd
}

func (r *TopicsCmd) Execute() error {
	c, err := sarama.NewConsumer(strings.Split(r.Brokers, ","), r.Config)
	if err != nil {
		return err
	}

	//get all topic from cluster
	topics, err := c.Topics()
	if err != nil {
		return err
	}

	for i, topic := range topics {
		fmt.Printf("Topic %d/%d: %s\n", i+1, len(topics), topic)
	}

	return nil
}
