package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricValue представляет структуру метрики из Kafka
type MetricValue struct {
	Type        string  `json:"Type"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Value       float64 `json:"Value"`
}

// MetricsData содержит все метрики из сообщения Kafka
type MetricsData map[string]MetricValue

var (
	metricsRegistry = prometheus.NewRegistry()
	gauges          = make(map[string]*prometheus.GaugeVec)
	counters        = make(map[string]*prometheus.CounterVec)
	metricsMutex    sync.RWMutex
)

func main() {
	// Получение параметров из переменных окружения
	bootstrapServers := getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	topic := getEnv("KAFKA_TOPIC", "metrics")
	groupID := getEnv("KAFKA_GROUP_ID", "prometheus-exporter")
	listenAddr := getEnv("LISTEN_ADDR", ":8080")

	// Настройка Kafka Consumer
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":       bootstrapServers,
		"group.id":                groupID,
		"auto.offset.reset":       "earliest",
		"enable.auto.commit":      true,
		"auto.commit.interval.ms": 5000,
		"session.timeout.ms":      6000,
		"max.poll.interval.ms":    300000,
		"statistics.interval.ms":  5000,
		"enable.partition.eof":    true,
	})

	if err != nil {
		log.Fatalf("Failed to create consumer: %s", err)
	}

	defer consumer.Close()

	// Подписка на топик
	err = consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		log.Fatalf("Failed to subscribe to topic: %s", err)
	}

	// Настройка HTTP сервера для Prometheus
	http.Handle("/metrics", promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	server := &http.Server{Addr: listenAddr}

	// Запуск HTTP сервера в отдельной горутине
	go func() {
		log.Printf("Starting Prometheus exporter on %s", listenAddr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Обработка сигналов для graceful shutdown
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Создание контекста для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск обработчика сообщений в отдельной горутине
	go processMessages(ctx, consumer)

	// Ожидание сигнала для завершения работы
	sig := <-sigchan
	log.Printf("Caught signal %v: terminating", sig)

	// Graceful shutdown
	timeout, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(timeout); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}
}

func processMessages(ctx context.Context, consumer *kafka.Consumer) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := consumer.ReadMessage(100 * time.Millisecond)
			if err != nil {
				// Игнорируем ошибки таймаута
				if !strings.Contains(err.Error(), "timed out") {
					log.Printf("Consumer error: %v", err)
				}
				continue
			}

			var metricsData MetricsData
			if err := json.Unmarshal(msg.Value, &metricsData); err != nil {
				log.Printf("Error unmarshalling metrics data: %v", err)
				continue
			}

			// Обработка полученных метрик
			processMetrics(metricsData)
		}
	}
}

func processMetrics(metricsData MetricsData) {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	for metricName, metricValue := range metricsData {
		switch metricValue.Type {
		case "gauge":
			// Создаем новый gauge, если он еще не существует
			if _, exists := gauges[metricName]; !exists {
				gauge := prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: sanitizeMetricName(metricValue.Name),
						Help: metricValue.Description,
					},
					[]string{},
				)
				metricsRegistry.MustRegister(gauge)
				gauges[metricName] = gauge
			}
			gauges[metricName].WithLabelValues().Set(metricValue.Value)

		case "counter":
			// Создаем новый counter, если он еще не существует
			if _, exists := counters[metricName]; !exists {
				counter := prometheus.NewCounterVec(
					prometheus.CounterOpts{
						Name: sanitizeMetricName(metricValue.Name),
						Help: metricValue.Description,
					},
					[]string{},
				)
				metricsRegistry.MustRegister(counter)
				counters[metricName] = counter
			}
			// Для счетчика устанавливаем абсолютное значение (в реальном приложении нужно
			// обрабатывать разницу между значениями)
			counters[metricName].WithLabelValues().Add(metricValue.Value)
		}
	}
}

// sanitizeMetricName приводит имя метрики к формату, совместимому с Prometheus
func sanitizeMetricName(name string) string {
	// Заменяем все недопустимые символы на подчеркивания
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "_")
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.ReplaceAll(result, "-", "_")

	// Удаляем любые специальные символы
	for _, c := range []string{"!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "+", "=", "{", "}", "[", "]", "\\", "|", ":", ";", "\"", "'", "<", ">", ",", "/"} {
		result = strings.ReplaceAll(result, c, "")
	}

	return result
}

// getEnv возвращает значение переменной окружения или default, если переменная не задана
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
