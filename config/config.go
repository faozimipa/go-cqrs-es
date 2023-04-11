package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/faozimipa/go-cqrs-es/pkg/constants"
	"github.com/faozimipa/go-cqrs-es/pkg/elastic"
	"github.com/faozimipa/go-cqrs-es/pkg/es"
	kafkaClient "github.com/faozimipa/go-cqrs-es/pkg/kafka"
	"github.com/faozimipa/go-cqrs-es/pkg/logger"
	"github.com/faozimipa/go-cqrs-es/pkg/migrations"
	"github.com/faozimipa/go-cqrs-es/pkg/mongodb"
	"github.com/faozimipa/go-cqrs-es/pkg/postgres"
	"github.com/faozimipa/go-cqrs-es/pkg/probes"
	"github.com/faozimipa/go-cqrs-es/pkg/tracing"
	"github.com/pkg/errors"

	"github.com/spf13/viper"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "", "BankAccount microservice config path")
}

type Config struct {
	ServiceName          string                  `mapstructure:"serviceName"`
	Logger               logger.LogConfig        `mapstructure:"logger"`
	GRPC                 GRPC                    `mapstructure:"grpc"`
	Postgresql           postgres.Config         `mapstructure:"postgres"`
	Timeouts             Timeouts                `mapstructure:"timeouts" validate:"required"`
	EventSourcingConfig  es.Config               `mapstructure:"eventSourcingConfig" validate:"required"`
	Kafka                *kafkaClient.Config     `mapstructure:"kafka" validate:"required"`
	KafkaTopics          KafkaTopics             `mapstructure:"kafkaTopics" validate:"required"`
	Mongo                *mongodb.Config         `mapstructure:"mongo" validate:"required"`
	MongoCollections     MongoCollections        `mapstructure:"mongoCollections" validate:"required"`
	KafkaPublisherConfig es.KafkaEventsBusConfig `mapstructure:"kafkaPublisherConfig" validate:"required"`
	Jaeger               *tracing.Config         `mapstructure:"jaeger"`
	ElasticIndexes       ElasticIndexes          `mapstructure:"elasticIndexes" validate:"required"`
	Projections          Projections             `mapstructure:"projections"`
	Http                 Http                    `mapstructure:"http"`
	Probes               probes.Config           `mapstructure:"probes"`
	ElasticSearch        elastic.Config          `mapstructure:"elasticSearch" validate:"required"`
	MigrationsConfig     migrations.Config       `mapstructure:"migrations" validate:"required"`
}

type GRPC struct {
	Port        string `mapstructure:"port"`
	Development bool   `mapstructure:"development"`
}

type Timeouts struct {
	PostgresInitMilliseconds int  `mapstructure:"postgresInitMilliseconds" validate:"required"`
	PostgresInitRetryCount   uint `mapstructure:"postgresInitRetryCount" validate:"required"`
}

type KafkaTopics struct {
	EventCreated                  kafkaClient.TopicConfig `mapstructure:"eventCreated" validate:"required"`
	BankAccountAggregateTypeTopic kafkaClient.TopicConfig `mapstructure:"orderAggregateTypeTopic" validate:"required"`
}

type MongoCollections struct {
	BankAccounts string `mapstructure:"bankAccounts" validate:"required"`
}

type ElasticIndexes struct {
	BankAccounts string `mapstructure:"bankAccounts" validate:"required"`
}

type Projections struct {
	MongoGroup                  string `mapstructure:"mongoGroup" validate:"required"`
	MongoSubscriptionPoolSize   int    `mapstructure:"mongoSubscriptionPoolSize" validate:"required,gte=0"`
	ElasticGroup                string `mapstructure:"elasticGroup" validate:"required"`
	ElasticSubscriptionPoolSize int    `mapstructure:"elasticSubscriptionPoolSize" validate:"required,gte=0"`
}

type Http struct {
	Port                string   `mapstructure:"port" validate:"required"`
	Development         bool     `mapstructure:"development"`
	BasePath            string   `mapstructure:"basePath" validate:"required"`
	BankAccountsPath    string   `mapstructure:"bankAccountsPath" validate:"required"`
	DebugErrorsResponse bool     `mapstructure:"debugErrorsResponse"`
	IgnoreLogUrls       []string `mapstructure:"ignoreLogUrls"`
}

func InitConfig() (*Config, error) {
	if configPath == "" {
		configPathFromEnv := os.Getenv(constants.ConfigPath)
		if configPathFromEnv != "" {
			configPath = configPathFromEnv
		} else {
			getwd, err := os.Getwd()
			if err != nil {
				return nil, errors.Wrap(err, "os.Getwd")
			}
			configPath = fmt.Sprintf("%s/config/config.yaml", getwd)
		}
	}

	cfg := &Config{}

	viper.SetConfigType(constants.Yaml)
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "viper.ReadInConfig")
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, errors.Wrap(err, "viper.Unmarshal")
	}

	grpcPort := os.Getenv(constants.GrpcPort)
	if grpcPort != "" {
		cfg.GRPC.Port = grpcPort
	}
	mongoURI := os.Getenv(constants.MongoDbURI)
	if mongoURI != "" {
		cfg.Mongo.URI = mongoURI
	}
	jaegerAddr := os.Getenv(constants.JaegerHostPort)
	if jaegerAddr != "" {
		cfg.Jaeger.HostPort = jaegerAddr
	}

	elasticUrl := os.Getenv(constants.ElasticUrl)
	if elasticUrl != "" {
		cfg.ElasticSearch.Addresses = []string{elasticUrl}
	}

	postgresHost := os.Getenv(constants.PostgresqlHost)
	if postgresHost != "" {
		cfg.Postgresql.Host = postgresHost
	}

	postgresPort := os.Getenv(constants.PostgresqlPort)
	if postgresPort != "" {
		cfg.Postgresql.Port = postgresPort
	}

	dbUrl := os.Getenv(constants.MIGRATIONS_DB_URL)
	if dbUrl != "" {
		cfg.MigrationsConfig.DbURL = dbUrl
	}

	kafkaBrokers := os.Getenv(constants.KafkaBrokers)
	if kafkaBrokers != "" {
		cfg.Kafka.Brokers = []string{kafkaBrokers}
	}

	return cfg, nil
}
