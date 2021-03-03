package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/spf13/cobra"
)

type HttpProducerCmd struct {
	KakfaConnect

	Addr   string `json:",omitempty"`
	CaFile string `json:",omitempty"`
}

func initHttpProducerCmd(root *cobra.Command) {
	c := HttpProducerCmd{}
	cmd := &cobra.Command{
		Use:   "http-producer",
		Short: "Produce a message by http query",
		Long: `Shows how to use the AsyncProducer and SyncProducer, and how to test them using mocks.
The server simply sends the data of the HTTP request's query string to Kafka,
and send a 200 result if that succeeds. For every request, it will send
an access log entry to Kafka as well in the background.`,
		Run: func(cmd *cobra.Command, args []string) { c.run() },
	}
	root.AddCommand(cmd)

	f := cmd.Flags()
	c.KakfaConnect.InitFlags(f)
	f.StringVar(&c.Addr, "addr", ":8080", "The address to bind to")
	f.StringVar(&c.CaFile, "ca", "", "The optional certificate authority file for TLS client authentication")
}

func (r *HttpProducerCmd) run() {
	if r.Verbose {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}

	if r.Brokers == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	brokerList := strings.Split(r.Brokers, ",")
	log.Printf("Kafka brokers: %s", strings.Join(brokerList, ", "))

	server := &Server{
		DataCollector:     r.newDataCollector(brokerList),
		AccessLogProducer: r.newAccessLogProducer(brokerList),
	}
	defer func() {
		if err := server.Close(); err != nil {
			log.Println("Failed to close server", err)
		}
	}()

	log.Fatal(server.Run(r.Addr))
}

func (r HttpProducerCmd) createTlsConfiguration() (t *tls.Config) {
	if r.TLSCertFile != "" && r.TlsKeyFile != "" && r.CaFile != "" {
		cert, err := tls.LoadX509KeyPair(r.TLSCertFile, r.TlsKeyFile)
		if err != nil {
			log.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(r.CaFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: r.TlsSkipVerify,
		}
	}

	// will be nil by default if nothing is provided
	return t
}

type Server struct {
	DataCollector     sarama.SyncProducer
	AccessLogProducer sarama.AsyncProducer
}

func (s *Server) Close() error {
	if err := s.DataCollector.Close(); err != nil {
		log.Println("Failed to shut down data collector cleanly", err)
	}

	if err := s.AccessLogProducer.Close(); err != nil {
		log.Println("Failed to shut down access log producer cleanly", err)
	}

	return nil
}

func (s *Server) Handler() http.Handler {
	return s.withAccessLog(s.collectQueryStringData())
}

func (s *Server) Run(addr string) error {
	httpServer := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	log.Printf("Listening for requests on %s...\n", addr)
	return httpServer.ListenAndServe()
}

func (s *Server) collectQueryStringData() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// We are not setting a message key, which means that all messages will
		// be distributed randomly over the different partitions.
		partition, offset, err := s.DataCollector.SendMessage(&sarama.ProducerMessage{
			Topic: "demo.http.rawquery",
			Value: sarama.StringEncoder(r.URL.RawQuery),
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to store your data:, %s", err)
		} else {
			// The tuple (topic, partition, offset) can be used as a unique identifier
			// for a message in a Kafka cluster.
			fmt.Fprintf(w, "Your data is stored with unique identifier important/%d/%d", partition, offset)
		}
	})
}

type accessLogEntry struct {
	Method       string  `json:"method"`
	Host         string  `json:"host"`
	Path         string  `json:"path"`
	IP           string  `json:"ip"`
	ResponseTime float64 `json:"response_time"`

	encoded []byte
	err     error
}

func (ale *accessLogEntry) ensureEncoded() {
	if ale.encoded == nil && ale.err == nil {
		ale.encoded, ale.err = json.Marshal(ale)
	}
}

func (ale *accessLogEntry) Length() int {
	ale.ensureEncoded()
	return len(ale.encoded)
}

func (ale *accessLogEntry) Encode() ([]byte, error) {
	ale.ensureEncoded()
	return ale.encoded, ale.err
}

func (s *Server) withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()

		next.ServeHTTP(w, r)

		entry := &accessLogEntry{
			Method:       r.Method,
			Host:         r.Host,
			Path:         r.RequestURI,
			IP:           r.RemoteAddr,
			ResponseTime: float64(time.Since(started)) / float64(time.Second),
		}

		// We will use the client's IP address as key. This will cause
		// all the access log entries of the same IP address to end up
		// on the same partition.
		s.AccessLogProducer.Input() <- &sarama.ProducerMessage{
			Topic: "demo.http.access_log",
			Key:   sarama.StringEncoder(r.RemoteAddr),
			Value: entry,
		}
	})
}

func (r HttpProducerCmd) newDataCollector(brokerList []string) sarama.SyncProducer {
	// For the data collector, we are looking for strong consistency semantics.
	// Because we don't change the flush settings, sarama will try to produce messages
	// as fast as possible to keep latency low.
	cnf := sarama.NewConfig()
	version, err := sarama.ParseKafkaVersion(r.Version)
	if err != nil {
		log.Panicf("Error parsing Kafka version: %v", err)
	}

	cnf.Version = version
	cnf.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	cnf.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	cnf.Producer.Return.Successes = true
	tlsConfig := r.createTlsConfiguration()
	if tlsConfig != nil {
		cnf.Net.TLS.Config = tlsConfig
		cnf.Net.TLS.Enable = true
	}

	// On the broker side, you may want to change the following settings to get
	// stronger consistency guarantees:
	// - For your broker, set `unclean.leader.election.enable` to false
	// - For the topic, you could increase `min.insync.replicas`.

	producer, err := sarama.NewSyncProducer(brokerList, cnf)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	return producer
}

func (r HttpProducerCmd) newAccessLogProducer(brokerList []string) sarama.AsyncProducer {
	// For the access log, we are looking for AP semantics, with high throughput.
	// By creating batches of compressed messages, we reduce network I/O at a cost of more latency.
	cnf := sarama.NewConfig()
	version, err := sarama.ParseKafkaVersion(r.Version)
	if err != nil {
		log.Panicf("Error parsing Kafka version: %v", err)
	}

	cnf.Version = version

	tlsConfig := r.createTlsConfiguration()
	if tlsConfig != nil {
		cnf.Net.TLS.Enable = true
		cnf.Net.TLS.Config = tlsConfig
	}
	cnf.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	cnf.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	cnf.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	producer, err := sarama.NewAsyncProducer(brokerList, cnf)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write access log entry:", err)
		}
	}()

	return producer
}
