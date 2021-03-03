package main

import "C"
import (
	"log"
	"os"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/tools/tls"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	initHttpProducerCmd(rootCmd)
	initConsumerCmd(rootCmd)
	initConsoleConsumerCmd(rootCmd)
	initProducerCmd(rootCmd)
	initAdminCmd(rootCmd)

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

func (r *KakfaConnect) InitFlags(f *pflag.FlagSet) *EnvPflag {
	e := NewEnvPflag("KAFKA", f)

	e.StringVarP(&r.Brokers, "brokers", "B", "127.0.0.1:9092",
		"The comma separated list of brokers in the Kafka cluster, or env KAFKA_BROKERS, or 127.0.0.1:9092")
	e.StringVarP(&r.Version, "version", "v", sarama.V0_10_2_1.String(), "Kafka cluster version")
	f.BoolVarP(&r.Verbose, "verbose", "V", false, "Whether to turn on sarama logging")
	f.BoolVar(&r.TlsSkipVerify, "tls-skip", false, "Whether skip TLS server cert verification")
	f.StringVar(&r.TLSCertFile, "tls-cert", "", "Cert for client authentication (use with -tls-key)")
	f.StringVar(&r.TlsKeyFile, "tls-key", "", "Key for client authentication (use with -tls-cert)")

	return e
}

type EnvPflag struct {
	Prefix  string
	FlagSet *pflag.FlagSet
	Env     map[string]string
}

func (f *EnvPflag) ReadEnv(name string) string {
	envKey := strings.ToUpper(name)
	if f.Prefix != "" {
		envKey = f.Prefix + "_" + envKey
	}

	envKey = strings.Replace(envKey, "-", "_", -1)
	return f.Env[envKey]
}

func NewEnvPflag(prefix string, f *pflag.FlagSet) *EnvPflag {
	environ := os.Environ()
	env := make(map[string]string)
	for _, s := range environ {
		if i := strings.Index(s, "="); i >= 1 {
			env[s[0:i]] = s[i+1:]
		}
	}

	if prefix != "" && strings.HasSuffix(prefix, "_") {
		prefix = prefix[:len(prefix)-1]
	}

	return &EnvPflag{
		Prefix:  prefix,
		FlagSet: f,
		Env:     env,
	}
}

func (f *EnvPflag) StringVar(s *string, name, value, usage string) {
	f.StringVarP(s, name, "", value, usage)
}

func (f *EnvPflag) StringVarP(s *string, name, shorthand, value, usage string) {
	if v := f.ReadEnv(name); v != "" {
		value = v
	}

	f.FlagSet.StringVarP(s, name, shorthand, value, usage)
}

func (f *EnvPflag) BoolVar(s *bool, name string, value bool, usage string) {
	f.BoolVarP(s, name, "", value, usage)
}

func (f *EnvPflag) BoolVarP(s *bool, name, shorthand string, value bool, usage string) {
	if v := f.ReadEnv(name); v != "" {
		value = v == "true" || v == "yes" || v == "y" || v == "t" || v == "on"
	}

	f.FlagSet.BoolVarP(s, name, shorthand, value, usage)
}

func (r *KakfaConnect) SetupVersion(config *sarama.Config) {
	version, err := sarama.ParseKafkaVersion(r.Version)
	if err != nil {
		log.Panicf("Error parsing Kafka version: %v", err)
	}

	config.Version = version
}

func (r *KakfaConnect) SetupTlSConfig(config *sarama.Config) {
	if r.TLSCertFile == "" || r.TlsKeyFile == "" {
		return
	}

	tlsConfig, err := tls.NewConfig(r.TLSCertFile, r.TlsKeyFile)
	if err != nil {
		printErrorAndExit(69, "Failed to create TLS config: %s", err)
	}

	config.Net.TLS.Enable = true
	config.Net.TLS.Config = tlsConfig
	config.Net.TLS.Config.InsecureSkipVerify = r.TlsSkipVerify
}
