using Confluent.Kafka;
using Microsoft.Extensions.Hosting;
using Microsoft.Extensions.Logging;
using System.Text.Json;

namespace Ecommerce.Shared.Kafka;

public interface IMessageHandler<T>
{
    Task HandleAsync(T message, CancellationToken cancellationToken);
}

public class KafkaConsumer<T> : BackgroundService
{
    private readonly IConsumer<string, string> _consumer;
    private readonly IMessageHandler<T> _messageHandler;
    private readonly ILogger<KafkaConsumer<T>> _logger;
    private readonly string _topic;

    public KafkaConsumer(
        KafkaConsumerConfig config,
        IMessageHandler<T> messageHandler,
        ILogger<KafkaConsumer<T>> logger)
    {
        var consumerConfig = new ConsumerConfig
        {
            BootstrapServers = string.Join(",", config.Brokers),
            GroupId = config.GroupId,
            AutoOffsetReset = config.AutoOffsetReset,
            EnableAutoCommit = false
        };

        _consumer = new ConsumerBuilder<string, string>(consumerConfig).Build();
        _messageHandler = messageHandler;
        _logger = logger;
        _topic = config.Topic;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _consumer.Subscribe(_topic);

        _logger.LogInformation("Kafka consumer started for topic: {Topic}", _topic);

        try
        {
            while (!stoppingToken.IsCancellationRequested)
            {
                try
                {
                    var consumeResult = _consumer.Consume(stoppingToken);

                    if (consumeResult?.Message == null)
                        continue;

                    _logger.LogInformation(
                        "Message received from Kafka. Topic: {Topic}, Partition: {Partition}, Offset: {Offset}",
                        consumeResult.Topic, consumeResult.Partition.Value, consumeResult.Offset.Value);

                    var message = JsonSerializer.Deserialize<T>(consumeResult.Message.Value);

                    if (message != null)
                    {
                        await _messageHandler.HandleAsync(message, stoppingToken);

                        // Commit offset after successful processing
                        _consumer.Commit(consumeResult);
                    }
                }
                catch (ConsumeException ex)
                {
                    _logger.LogError(ex, "Error consuming message from Kafka");
                }
                catch (JsonException ex)
                {
                    _logger.LogError(ex, "Error deserializing message from Kafka");
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Error handling message from Kafka");
                }
            }
        }
        catch (OperationCanceledException)
        {
            _logger.LogInformation("Kafka consumer stopped");
        }
        finally
        {
            _consumer.Close();
            _consumer.Dispose();
        }
    }

    public override void Dispose()
    {
        _consumer?.Dispose();
        base.Dispose();
    }
}

public class KafkaConsumerConfig
{
    public List<string> Brokers { get; set; } = new();
    public string Topic { get; set; } = string.Empty;
    public string GroupId { get; set; } = string.Empty;
    public AutoOffsetReset AutoOffsetReset { get; set; } = AutoOffsetReset.Earliest;
}
