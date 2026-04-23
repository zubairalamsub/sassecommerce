using Confluent.Kafka;
using System.Text.Json;

namespace Ecommerce.InventoryService.Messaging;

public interface IEventPublisher
{
    Task PublishAsync(string eventType, Dictionary<string, object> payload, CancellationToken cancellationToken = default);
}

public class KafkaEventPublisher : IEventPublisher, IDisposable
{
    private readonly IProducer<string, string> _producer;
    private readonly ILogger<KafkaEventPublisher> _logger;
    private const string Topic = "inventory-events";

    public KafkaEventPublisher(IConfiguration configuration, ILogger<KafkaEventPublisher> logger)
    {
        var brokers = configuration["KAFKA_BROKERS"] ?? "kafka:9092";

        var config = new ProducerConfig
        {
            BootstrapServers = brokers,
            Acks = Acks.Leader,
            CompressionType = CompressionType.Snappy
        };

        _producer = new ProducerBuilder<string, string>(config).Build();
        _logger = logger;

        _logger.LogInformation("Kafka producer initialized for topic {Topic}, brokers: {Brokers}", Topic, brokers);
    }

    public async Task PublishAsync(string eventType, Dictionary<string, object> payload, CancellationToken cancellationToken = default)
    {
        var eventId = Guid.NewGuid().ToString();

        var envelope = new Dictionary<string, object>
        {
            ["event_id"] = eventId,
            ["event_type"] = eventType,
            ["timestamp"] = DateTime.UtcNow.ToString("o"),
            ["version"] = "1.0.0",
            ["payload"] = payload
        };

        try
        {
            var json = JsonSerializer.Serialize(envelope);
            var message = new Message<string, string> { Key = eventId, Value = json };

            await _producer.ProduceAsync(Topic, message, cancellationToken);

            _logger.LogDebug("Published {EventType} event to {Topic}", eventType, Topic);
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Failed to publish {EventType} event to Kafka", eventType);
        }
    }

    public void Dispose()
    {
        _producer?.Flush(TimeSpan.FromSeconds(5));
        _producer?.Dispose();
    }
}
