using Confluent.Kafka;
using Microsoft.Extensions.Logging;
using System.Text.Json;

namespace Ecommerce.Shared.Kafka;

public interface IKafkaProducer
{
    Task PublishAsync<T>(string topic, string key, T value, CancellationToken cancellationToken = default);
    Task PublishAsync<T>(string topic, string key, T value, Dictionary<string, string> headers, CancellationToken cancellationToken = default);
}

public class KafkaProducer : IKafkaProducer, IDisposable
{
    private readonly IProducer<string, string> _producer;
    private readonly ILogger<KafkaProducer> _logger;

    public KafkaProducer(KafkaProducerConfig config, ILogger<KafkaProducer> logger)
    {
        var producerConfig = new ProducerConfig
        {
            BootstrapServers = string.Join(",", config.Brokers),
            Acks = (Acks)config.RequiredAcks,
            CompressionType = CompressionType.Snappy,
            LingerMs = 10,
            BatchSize = 100000
        };

        _producer = new ProducerBuilder<string, string>(producerConfig).Build();
        _logger = logger;
    }

    public async Task PublishAsync<T>(string topic, string key, T value, CancellationToken cancellationToken = default)
    {
        await PublishAsync(topic, key, value, new Dictionary<string, string>(), cancellationToken);
    }

    public async Task PublishAsync<T>(
        string topic,
        string key,
        T value,
        Dictionary<string, string> headers,
        CancellationToken cancellationToken = default)
    {
        try
        {
            var jsonValue = JsonSerializer.Serialize(value);

            var message = new Message<string, string>
            {
                Key = key,
                Value = jsonValue,
                Timestamp = Timestamp.Default
            };

            // Add headers
            if (headers.Any())
            {
                message.Headers = new Headers();
                foreach (var header in headers)
                {
                    message.Headers.Add(header.Key, System.Text.Encoding.UTF8.GetBytes(header.Value));
                }
            }

            var result = await _producer.ProduceAsync(topic, message, cancellationToken);

            _logger.LogInformation(
                "Message published to Kafka. Topic: {Topic}, Partition: {Partition}, Offset: {Offset}",
                result.Topic, result.Partition.Value, result.Offset.Value);
        }
        catch (ProduceException<string, string> ex)
        {
            _logger.LogError(ex, "Error publishing message to Kafka. Topic: {Topic}, Key: {Key}", topic, key);
            throw;
        }
    }

    public void Dispose()
    {
        _producer?.Flush(TimeSpan.FromSeconds(10));
        _producer?.Dispose();
    }
}

public class KafkaProducerConfig
{
    public List<string> Brokers { get; set; } = new();
    public int RequiredAcks { get; set; } = 1;
}
