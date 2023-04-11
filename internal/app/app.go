package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/faozimipa/go-cqrs-es/config"
	v1 "github.com/faozimipa/go-cqrs-es/internal/bankAccount/delivery/http/v1"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/delivery/kafka/elasticsearch_subscription"
	bankAccountMongoSubscription "github.com/faozimipa/go-cqrs-es/internal/bankAccount/delivery/kafka/mongo_subscription"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/domain"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/projection/elasticsearch_projection"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/projection/mongo_projection"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/repository/elasticsearch_repository"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/repository/mongo_repository"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/service"
	"github.com/faozimipa/go-cqrs-es/internal/metrics"
	"github.com/faozimipa/go-cqrs-es/pkg/elastic"
	"github.com/faozimipa/go-cqrs-es/pkg/es"
	"github.com/faozimipa/go-cqrs-es/pkg/esclient"
	"github.com/faozimipa/go-cqrs-es/pkg/interceptors"
	kafkaClient "github.com/faozimipa/go-cqrs-es/pkg/kafka"
	"github.com/faozimipa/go-cqrs-es/pkg/logger"
	"github.com/faozimipa/go-cqrs-es/pkg/middlewares"
	"github.com/faozimipa/go-cqrs-es/pkg/mongodb"
	"github.com/faozimipa/go-cqrs-es/pkg/tracing"
	"github.com/go-playground/validator"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	maxConnectionIdle = 5
	gRPCTimeout       = 15
	maxConnectionAge  = 5
	gRPCTime          = 10
)

type app struct {
	log                logger.Logger
	cfg                config.Config
	interceptorManager interceptors.InterceptorManager
	middlewareManager  middlewares.MiddlewareManager
	probesSrv          *http.Server
	validate           *validator.Validate
	metrics            *metrics.ESMicroserviceMetrics
	kafkaConn          *kafka.Conn
	pgxConn            *pgxpool.Pool
	mongoClient        *mongo.Client
	doneCh             chan struct{}
	elasticClient      *elasticsearch.Client
	echo               *echo.Echo
	probeServer        *http.Server
	bankAccountService *service.BankAccountService
}

func NewApp(log logger.Logger, cfg config.Config) *app {
	return &app{log: log, cfg: cfg, validate: validator.New(), doneCh: make(chan struct{}), echo: echo.New()}
}

func (a *app) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// enable tracing
	if a.cfg.Jaeger.Enable {
		tracer, closer, err := tracing.NewJaegerTracer(a.cfg.Jaeger)
		if err != nil {
			return err
		}
		defer closer.Close() // nolint: errcheck
		opentracing.SetGlobalTracer(tracer)
	}

	a.metrics = metrics.NewESMicroserviceMetrics(a.cfg)
	a.interceptorManager = interceptors.NewInterceptorManager(a.log, a.getGrpcMetricsCb())
	a.middlewareManager = middlewares.NewMiddlewareManager(a.log, a.cfg, a.getHttpMetricsCb())

	// connect postgres
	if err := a.connectPostgres(ctx); err != nil {
		return err
	}
	defer a.pgxConn.Close()

	if err := a.runMigrate(); err != nil {
		return err
	}

	// connect mongo
	mongoDBConn, err := mongodb.NewMongoDBConn(ctx, a.cfg.Mongo)
	if err != nil {
		return errors.Wrap(err, "NewMongoDBConn")
	}
	a.mongoClient = mongoDBConn
	defer mongoDBConn.Disconnect(ctx) // nolint: errcheck
	a.log.Infof("(Mongo connected) SessionsInProgress: %v", mongoDBConn.NumberSessionsInProgress())

	a.initMongoDBCollections(ctx)

	elasticSearchClient, err := elastic.NewElasticSearchClient(a.cfg.ElasticSearch)
	if err != nil {
		return err
	}
	a.elasticClient = elasticSearchClient

	// connect elastic
	elasticInfoResponse, err := esclient.Info(ctx, a.elasticClient)
	if err != nil {
		return err
	}
	a.log.Infof("Elastic info response: %s", elasticInfoResponse.String())

	// connect kafka brokers
	if err := a.connectKafkaBrokers(ctx); err != nil {
		return errors.Wrap(err, "a.connectKafkaBrokers")
	}
	defer a.kafkaConn.Close() // nolint: errcheck

	// init kafka topics
	if a.cfg.Kafka.InitTopics {
		a.initKafkaTopics(ctx)
	}

	// kafka producer
	kafkaProducer := kafkaClient.NewProducer(a.log, a.cfg.Kafka.Brokers)
	defer kafkaProducer.Close() // nolint: errcheck

	eventSerializer := domain.NewEventSerializer()
	eventBus := es.NewKafkaEventsBus(kafkaProducer, a.cfg.KafkaPublisherConfig)
	eventStore := es.NewPgEventStore(a.log, a.cfg.EventSourcingConfig, a.pgxConn, eventBus, eventSerializer)

	mongoRepository := mongo_repository.NewBankAccountMongoRepository(a.log, &a.cfg, a.mongoClient)
	mongoProjection := mongo_projection.NewBankAccountMongoProjection(a.log, &a.cfg, eventSerializer, mongoRepository)

	elasticSearchRepository := elasticsearch_repository.NewElasticRepository(a.log, &a.cfg, a.elasticClient)
	elasticSearchProjection := elasticsearch_projection.NewElasticProjection(a.log, &a.cfg, eventSerializer, elasticSearchRepository)

	a.bankAccountService = service.NewBankAccountService(a.log, eventStore, mongoRepository, elasticSearchRepository)

	v1.NewBankAccountHandlers(a.echo.Group(a.cfg.Http.BankAccountsPath), a.middlewareManager, a.log, &a.cfg, a.bankAccountService, a.validate, a.metrics).MapRoutes()

	mongoSubscription := bankAccountMongoSubscription.NewBankAccountMongoSubscription(
		a.log,
		&a.cfg,
		a.bankAccountService,
		mongoProjection,
		eventSerializer,
		mongoRepository,
		eventStore,
		eventBus,
	)

	mongoConsumerGroup := kafkaClient.NewConsumerGroup(a.cfg.Kafka.Brokers, a.cfg.Projections.MongoGroup, a.log)
	go func() {
		err := mongoConsumerGroup.ConsumeTopicWithErrGroup(
			ctx,
			a.getConsumerGroupTopics(),
			a.cfg.Projections.MongoSubscriptionPoolSize,
			mongoSubscription.ProcessMessagesErrGroup,
		)
		if err != nil {
			a.log.Errorf("(mongoConsumerGroup ConsumeTopicWithErrGroup) err: %v", err)
			cancel()
			return
		}
	}()

	elasticSearchSubscription := elasticsearch_subscription.NewElasticSearchSubscription(
		a.log,
		&a.cfg,
		a.bankAccountService,
		elasticSearchProjection,
		eventSerializer,
		elasticSearchRepository,
		eventStore,
		eventBus,
	)

	elasticSearchConsumerGroup := kafkaClient.NewConsumerGroup(a.cfg.Kafka.Brokers, a.cfg.Projections.ElasticGroup, a.log)
	go func() {
		err := elasticSearchConsumerGroup.ConsumeTopicWithErrGroup(
			ctx,
			a.getConsumerGroupTopics(),
			a.cfg.Projections.ElasticSubscriptionPoolSize,
			elasticSearchSubscription.ProcessMessagesErrGroup,
		)
		if err != nil {
			a.log.Errorf("(elasticSearchConsumerGroup ConsumeTopicWithErrGroup) err: %v", err)
			cancel()
			return
		}
	}()

	closeGrpcServer, grpcServer, err := a.newBankAccountGrpcServer()
	if err != nil {
		cancel()
		return err
	}
	defer closeGrpcServer() // nolint: errcheck

	// run metrics and health check
	a.runMetrics(cancel)
	a.runHealthCheck(ctx)

	go func() {
		if err := a.runHttpServer(); err != nil {
			a.log.Errorf("(runHttpServer) err: %v", err)
			cancel()
		}
	}()
	a.log.Infof("%s is listening on PORT: %v", GetMicroserviceName(a.cfg), a.cfg.Http.Port)

	<-ctx.Done()
	a.waitShootDown(waitShotDownDuration)
	grpcServer.GracefulStop()
	if err := a.shutDownHealthCheckServer(ctx); err != nil {
		a.log.Warnf("(shutDownHealthCheckServer) err: %v", err)
	}

	if err := a.echo.Shutdown(ctx); err != nil {
		a.log.Warnf("(Shutdown) err: %v", err)
	}

	<-a.doneCh
	a.log.Infof("%s app exited properly", GetMicroserviceName(a.cfg))
	return nil
}
