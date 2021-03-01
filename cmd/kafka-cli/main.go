package main

import "C"
import (
	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/tools/tls"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"log"
	"os"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kafka-cli",
		Short: "client utilities for kafka",
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of my",
		Run: func(cmd *cobra.Command, args []string) {
			log.Print("0.1")
		},
	})
	initProducerCmd(rootCmd)
	initConsumerCmd(rootCmd)
	initConsoleConsumerCmd(rootCmd)
	initConsoleProducerCmd(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Printf("E! error: %v", err)
		os.Exit(1)
	}
}

type KakfaConnect struct {
	Brokers string `json:",omitempty"`
	Version string `json:",omitempty"`
	Verbose bool   `json:",omitempty"`

	TlsSkipVerify bool   `json:",omitempty"`
	TLSCertFile   string `json:",omitempty"`
	TlsKeyFile    string `json:",omitempty"`
}

func (r *KakfaConnect) InitFlags(f *pflag.FlagSet) {
	kafkaPeers := os.Getenv("KAFKA_PEERS")
	if kafkaPeers == "" {
		kafkaPeers = "127.0.0.1:9092"
	}

	f.StringVar(&r.Brokers, "brokers", kafkaPeers, "The comma separated list of brokers in the Kafka cluster")
	f.StringVar(&r.Version, "version", sarama.V0_10_0_0.String(), "Kafka cluster version")
	f.BoolVar(&r.Verbose, "verbose", false, "Whether to turn on sarama logging")
	f.BoolVar(&r.TlsSkipVerify, "tls-skip", false, "Whether skip TLS server cert verification")
	f.StringVar(&r.TLSCertFile, "tls-cert", "", "Cert for client authentication (use with -tls-key)")
	f.StringVar(&r.TlsKeyFile, "tls-key", "", "Key for client authentication (use with -tls-cert)")
}

func (r *KakfaConnect) SetupTlSConfig(config *sarama.Config) {
	if r.TLSCertFile != "" && r.TlsKeyFile != "" {
		tlsConfig, err := tls.NewConfig(r.TLSCertFile, r.TlsKeyFile)
		if err != nil {
			printErrorAndExit(69, "Failed to create TLS config: %s", err)
		}

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Config.InsecureSkipVerify = r.TlsSkipVerify
	}
}
