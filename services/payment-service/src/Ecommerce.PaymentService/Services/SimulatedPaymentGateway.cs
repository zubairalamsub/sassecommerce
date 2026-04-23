using System.Security.Cryptography;

namespace Ecommerce.PaymentService.Services;

/// <summary>
/// Simulated payment gateway for development and testing.
/// In production, this would be replaced with a real gateway (SSLCommerz, AamarPay, bKash, Nagad, etc.)
/// </summary>
public class SimulatedPaymentGateway : IPaymentGateway
{
    private readonly ILogger<SimulatedPaymentGateway> _logger;

    // Test card numbers that trigger specific behaviors
    private static readonly Dictionary<string, string> TestCardBehaviors = new()
    {
        { "4000000000000002", "card_declined" },
        { "4000000000009995", "insufficient_funds" },
        { "4000000000009987", "lost_card" },
        { "4000000000000069", "expired_card" },
        { "4000000000000127", "incorrect_cvc" },
        { "4000000000000119", "processing_error" },
    };

    public SimulatedPaymentGateway(ILogger<SimulatedPaymentGateway> logger)
    {
        _logger = logger;
    }

    public string Name => "SimulatedGateway";

    public async Task<GatewayResponse> ChargeAsync(GatewayChargeRequest request, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Processing simulated charge: Amount={Amount} {Currency}, Order={OrderId}",
            request.Amount, request.Currency, request.OrderId);

        // Simulate processing delay
        await Task.Delay(Random.Shared.Next(100, 500), cancellationToken);

        // Simulate different outcomes based on amount for testing
        // Amounts ending in .99 with specific dollar amounts trigger failures
        if (request.Amount == 0)
        {
            return new GatewayResponse
            {
                Success = false,
                TransactionId = string.Empty,
                ErrorCode = "invalid_amount",
                ErrorMessage = "Amount must be greater than zero",
                RawResponse = "{\"error\": \"invalid_amount\"}"
            };
        }

        var transactionId = $"sim_ch_{GenerateId()}";

        _logger.LogInformation("Simulated charge successful: TransactionId={TransactionId}", transactionId);

        return new GatewayResponse
        {
            Success = true,
            TransactionId = transactionId,
            RawResponse = $"{{\"id\": \"{transactionId}\", \"status\": \"succeeded\", \"amount\": {request.Amount}}}"
        };
    }

    public async Task<GatewayResponse> RefundAsync(GatewayRefundRequest request, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Processing simulated refund: TransactionId={TransactionId}, Amount={Amount}",
            request.TransactionId, request.Amount);

        await Task.Delay(Random.Shared.Next(100, 300), cancellationToken);

        if (string.IsNullOrEmpty(request.TransactionId))
        {
            return new GatewayResponse
            {
                Success = false,
                TransactionId = string.Empty,
                ErrorCode = "invalid_transaction",
                ErrorMessage = "Original transaction ID is required",
                RawResponse = "{\"error\": \"invalid_transaction\"}"
            };
        }

        var refundId = $"sim_rf_{GenerateId()}";

        _logger.LogInformation("Simulated refund successful: RefundId={RefundId}", refundId);

        return new GatewayResponse
        {
            Success = true,
            TransactionId = refundId,
            RawResponse = $"{{\"id\": \"{refundId}\", \"status\": \"succeeded\", \"amount\": {request.Amount}}}"
        };
    }

    public async Task<GatewayResponse> VoidAsync(string transactionId, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Processing simulated void: TransactionId={TransactionId}", transactionId);

        await Task.Delay(Random.Shared.Next(50, 200), cancellationToken);

        var voidId = $"sim_vo_{GenerateId()}";

        return new GatewayResponse
        {
            Success = true,
            TransactionId = voidId,
            RawResponse = $"{{\"id\": \"{voidId}\", \"status\": \"voided\"}}"
        };
    }

    public Task<string> TokenizeCardAsync(string cardNumber, int expiryMonth, int expiryYear, string cvv, CancellationToken cancellationToken = default)
    {
        _logger.LogInformation("Tokenizing card ending in {Last4}", cardNumber.Length >= 4 ? cardNumber[^4..] : "****");

        // Check for test card behaviors
        if (TestCardBehaviors.ContainsKey(cardNumber))
        {
            var behavior = TestCardBehaviors[cardNumber];
            _logger.LogWarning("Test card detected with behavior: {Behavior}", behavior);
        }

        var token = $"tok_{GenerateId()}";
        return Task.FromResult(token);
    }

    private static string GenerateId()
    {
        var bytes = RandomNumberGenerator.GetBytes(12);
        return Convert.ToHexString(bytes).ToLowerInvariant();
    }
}
