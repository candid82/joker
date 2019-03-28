(import org.apache.kafka.clients.producer.KafkaProducer)

(proxy [KafkaProducer] []
  (send [producer-record] :whatever))

(proxy [String] []
  (toString []
    (println this)))


