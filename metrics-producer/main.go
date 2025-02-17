package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type MetricValue struct {
	Type        string  `json:"Type"`
	Name        string  `json:"Name"`
	Description string  `json:"Description"`
	Value       float64 `json:"Value"`
}

type MetricsData map[string]MetricValue

func main() {
	// Получение параметров из переменных окружения
	bootstrapServers := getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092")
	topic := getEnv("KAFKA_TOPIC", "metrics")

	// Настройка Kafka Producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		log.Fatalf("Failed to create producer: %s", err)
	}
	defer producer.Close()

	// Обработка событий доставки в отдельной горутине
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message: %v\n", ev.TopicPartition.Error)
				} else {
					log.Printf("Successfully produced message to topic %s\n", *ev.TopicPartition.Topic)
				}
			}
		}
	}()

	var pollCount float64 = 0

	// Генерация метрик каждые 10 секунд
	for {
		pollCount++

		// Создание тестовых метрик
		metrics := MetricsData{
			"Alloc": {
				Type:        "gauge",
				Name:        "Alloc",
				Description: "Alloc is bytes of allocated heap objects.",
				Value:       float64(20000000 + (pollCount * 1000)),
			},
			"FreeMemory": {
				Type:        "gauge",
				Name:        "FreeMemory",
				Description: "RAM available for programs to allocate",
				Value:       7740977152 - (pollCount * 10000),
			},
			"PollCount": {
				Type:        "counter",
				Name:        "PollCount",
				Description: "PollCount is quantity of metrics collection iteration.",
				Value:       pollCount,
			},
			"TotalMemory": {
				Type:        "gauge",
				Name:        "TotalMemory",
				Description: "Total amount of RAM on this system",
				Value:       16054480896,
			},
		}

		// Сериализация метрик в JSON
		metricsJSON, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("Error marshalling metrics: %v", err)
			continue
		}

		// Отправка метрик в Kafka
		message := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          metricsJSON,
		}

		err = producer.Produce(message, nil)
		if err != nil {
			log.Printf("Error producing message: %v", err)
		}

		time.Sleep(10 * time.Second)
	}
}

// getEnv возвращает значение переменной окружения или default, если переменная не задана
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
