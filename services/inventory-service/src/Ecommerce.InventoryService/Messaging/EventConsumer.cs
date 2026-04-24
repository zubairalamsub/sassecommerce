using System.Text.Json;
using Confluent.Kafka;
using Ecommerce.InventoryService.DTOs;
using Ecommerce.InventoryService.Services;

namespace Ecommerce.InventoryService.Messaging;

public class OrderEventConsumer : BackgroundService
{
    private readonly IServiceScopeFactory _scopeFactory;
    private readonly ILogger<OrderEventConsumer> _logger;
    private readonly IConsumer<string, string> _consumer;
    private const string Topic = "order-events";
    private const string GroupId = "inventory-service";

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

            using var scope = _scopeFactory.CreateScope();
            var inventoryService = scope.ServiceProvider.GetRequiredService<IInventoryService>();

            switch (envelope.EventType)
            {
                case "OrderPlaced":
                case "OrderCreated":
                    await HandleOrderPlaced(inventoryService, envelope, cancellationToken);
                    break;

                case "OrderCancelled":
                    await HandleOrderCancelled(inventoryService, envelope, cancellationToken);
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

    private async Task HandleOrderPlaced(IInventoryService inventoryService, EventEnvelope envelope, CancellationToken cancellationToken)
    {
        var payload = envelope.GetPayload();
        if (payload == null) return;

        var tenantId = GetString(payload, "tenant_id");
        var orderId = GetString(payload, "order_id");
        var userId = GetString(payload, "user_id");

        if (string.IsNullOrEmpty(tenantId) || string.IsNullOrEmpty(orderId))
        {
            _logger.LogWarning("OrderPlaced event missing tenant_id or order_id");
            return;
        }

        _logger.LogInformation("Processing OrderPlaced for order {OrderId}, tenant {TenantId}", orderId, tenantId);

        // Extract order items and reserve stock for each
        var items = GetItems(payload);
        foreach (var item in items)
        {
            try
            {
                var request = new ReserveStockRequest
                {
                    TenantId = tenantId,
                    ProductId = item.ProductId,
                    VariantId = item.VariantId,
                    Quantity = item.Quantity,
                    OrderId = orderId,
                    OrderItemId = item.OrderItemId ?? orderId,
                    ExpirationMinutes = 30,
                    CreatedBy = userId ?? "order-service"
                };

                await inventoryService.ReserveStockAsync(request, cancellationToken);
                _logger.LogInformation("Reserved {Quantity} of product {ProductId} for order {OrderId}",
                    item.Quantity, item.ProductId, orderId);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Failed to reserve stock for product {ProductId} in order {OrderId}",
                    item.ProductId, orderId);
            }
        }
    }

    private async Task HandleOrderCancelled(IInventoryService inventoryService, EventEnvelope envelope, CancellationToken cancellationToken)
    {
        var payload = envelope.GetPayload();
        if (payload == null) return;

        var tenantId = GetString(payload, "tenant_id");
        var orderId = GetString(payload, "order_id");
        var reason = GetString(payload, "reason") ?? "Order cancelled";

        if (string.IsNullOrEmpty(orderId))
        {
            _logger.LogWarning("OrderCancelled event missing order_id");
            return;
        }

        _logger.LogInformation("Processing OrderCancelled for order {OrderId}", orderId);

        // Find and cancel all stock reservations for this order
        try
        {
            var movements = await inventoryService.GetStockMovementsByOrderAsync(orderId, cancellationToken);
            _logger.LogInformation("Found {Count} stock movements for cancelled order {OrderId}", movements.Count, orderId);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to release stock for cancelled order {OrderId}", orderId);
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

    private static List<OrderItemInfo> GetItems(Dictionary<string, object> payload)
    {
        var items = new List<OrderItemInfo>();

        if (payload.TryGetValue("items", out var itemsObj) && itemsObj is JsonElement itemsElement)
        {
            if (itemsElement.ValueKind == JsonValueKind.Array)
            {
                foreach (var item in itemsElement.EnumerateArray())
                {
                    items.Add(new OrderItemInfo
                    {
                        ProductId = item.TryGetProperty("product_id", out var pid) ? pid.GetString() ?? "" : "",
                        VariantId = item.TryGetProperty("variant_id", out var vid) ? vid.GetString() : null,
                        Quantity = item.TryGetProperty("quantity", out var qty) ? qty.GetInt32() : 0,
                        OrderItemId = item.TryGetProperty("order_item_id", out var oiid) ? oiid.GetString() : null
                    });
                }
            }
        }

        // Fallback: single product in payload (flat structure)
        if (items.Count == 0)
        {
            var productId = GetString(payload, "product_id");
            if (!string.IsNullOrEmpty(productId))
            {
                var quantity = 1;
                if (payload.TryGetValue("quantity", out var qtyObj) && qtyObj is JsonElement qtyEl)
                    quantity = qtyEl.GetInt32();

                items.Add(new OrderItemInfo
                {
                    ProductId = productId,
                    VariantId = GetString(payload, "variant_id"),
                    Quantity = quantity,
                    OrderItemId = GetString(payload, "order_item_id")
                });
            }
        }

        return items;
    }

    public override void Dispose()
    {
        _consumer?.Dispose();
        base.Dispose();
    }

    private class OrderItemInfo
    {
        public string ProductId { get; set; } = string.Empty;
        public string? VariantId { get; set; }
        public int Quantity { get; set; }
        public string? OrderItemId { get; set; }
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
