using System.Text.Json;
using Confluent.Kafka;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Services;

namespace Ecommerce.PaymentService.Messaging;

public class OrderEventConsumer : BackgroundService
{
    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<OrderEventConsumer> _logger;
    private readonly IConsumer<string, string> _consumer;
    private const string Topic = "order-events";
    private const string GroupId = "payment-service";

    public OrderEventConsumer(IConfiguration configuration, IServiceScopeFactory scopeFactory, ILogger<OrderEventConsumer> logger)
    {
        _scopeFactory = scopeFactory;
        _logger = logger;

        var brokers = configuration["KAFKA_BROKERS"] ?? "kafka:9092";
        var config = new ConsumerConfig
        {
            BootstrapServers = brokers,
            GroupId = GroupId,
            AutoOffsetReset = AutoOffsetReset.Earliest,
            EnableAutoCommit = true,
            AutoCommitIntervalMs = 1000
        };

        _consumer = new ConsumerBuilder<string, string>(config).Build();
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("Order event consumer starting, subscribing to {Topic}", Topic);
        _consumer.Subscribe(Topic);

        await Task.Run(() => ConsumeLoop(stoppingToken), stoppingToken);
    }

    private void ConsumeLoop(CancellationToken stoppingToken)
    {
        try
        {
            while (!stoppingToken.IsCancellationRequested)
            {
                try
                {
                    var result = _consumer.Consume(stoppingToken);
                    if (result?.Message?.Value == null) continue;

                    ProcessMessage(result.Message.Value, stoppingToken).GetAwaiter().GetResult();
                }
                catch (ConsumeException ex)
                {
                    _logger.LogWarning(ex, "Error consuming message from {Topic}", Topic);
                }
            }
        }
        catch (OperationCanceledException)
        {
            _logger.LogInformation("Order event consumer shutting down");
        }
        finally
        {
            _consumer.Close();
        }
    }

    private async Task ProcessMessage(string messageValue, CancellationToken cancellationToken)
    {
        try
        {
            var envelope = JsonSerializer.Deserialize<EventEnvelope>(messageValue, new JsonSerializerOptions
            {
                PropertyNameCaseInsensitive = true
            });

            if (envelope == null)
            {
                _logger.LogWarning("Failed to deserialize event envelope");
                return;
            }

            _logger.LogDebug("Processing {EventType} event (ID: {EventId})", envelope.EventType, envelope.EventId);

            switch (envelope.EventType)
            {
                case "OrderPlaced":
                case "OrderCreated":
                    await HandleOrderPlaced(envelope, cancellationToken);
                    break;

                default:
                    _logger.LogDebug("Ignoring unhandled event type: {EventType}", envelope.EventType);
                    break;
            }
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Error processing order event");
        }
    }

    private async Task HandleOrderPlaced(EventEnvelope envelope, CancellationToken cancellationToken)
    {
        var payload = envelope.GetPayload();
        if (payload == null) return;

        var tenantId = GetString(payload, "tenant_id");
        var orderId = GetString(payload, "order_id");
        var userId = GetString(payload, "user_id");
        var amount = GetDecimal(payload, "total_amount") ?? GetDecimal(payload, "amount") ?? 0;
        var currency = GetString(payload, "currency") ?? "BDT";
        var paymentMethod = GetString(payload, "payment_method") ?? "bkash";

        if (string.IsNullOrEmpty(tenantId) || string.IsNullOrEmpty(orderId) || amount <= 0)
        {
            _logger.LogWarning("OrderPlaced event missing required fields (tenant_id, order_id, amount)");
            return;
        }

        _logger.LogInformation("Processing OrderPlaced for order {OrderId}, amount {Amount} {Currency}",
            orderId, amount, currency);

        using var scope = _scopeFactory.CreateScope();
        var paymentService = scope.ServiceProvider.GetRequiredService<IPaymentService>();

        try
        {
            var request = new CreatePaymentRequest
            {
                TenantId = tenantId,
                CustomerId = userId ?? "",
                OrderId = orderId,
                Amount = amount,
                Currency = currency,
                Method = paymentMethod,
                Description = $"Payment for order {orderId}",
                IdempotencyKey = $"order-{orderId}",
                CreatedBy = "order-event-consumer"
            };

            var result = await paymentService.ProcessPaymentAsync(request, cancellationToken);
            _logger.LogInformation("Payment {PaymentId} initiated for order {OrderId}, status: {Status}",
                result.Id, orderId, result.Status);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to process payment for order {OrderId}", orderId);
        }
    }

    private static string? GetString(Dictionary<string, object> payload, string key)
    {
        if (payload.TryGetValue(key, out var value))
        {
            if (value is JsonElement je)
                return je.GetString();
            return value?.ToString();
        }
        return null;
    }

    private static decimal? GetDecimal(Dictionary<string, object> payload, string key)
    {
        if (payload.TryGetValue(key, out var value))
        {
            if (value is JsonElement je)
            {
                if (je.ValueKind == JsonValueKind.Number)
                    return je.GetDecimal();
                if (je.ValueKind == JsonValueKind.String && decimal.TryParse(je.GetString(), out var parsed))
                    return parsed;
            }
            if (decimal.TryParse(value?.ToString(), out var d))
                return d;
        }
        return null;
    }

    public override void Dispose()
    {
        _consumer?.Dispose();
        base.Dispose();
    }
}

public class EventEnvelope
{
    public string EventId { get; set; } = string.Empty;
    public string EventType { get; set; } = string.Empty;
    public string Timestamp { get; set; } = string.Empty;
    public string? Version { get; set; }
    public Dictionary<string, object>? Payload { get; set; }
    public Dictionary<string, object>? Data { get; set; }

    public Dictionary<string, object>? GetPayload()
    {
        return Payload ?? Data;
    }
}
