namespace Ecommerce.PaymentService.Services;

public class GatewayChargeRequest
{
    public decimal Amount { get; set; }
    public string Currency { get; set; } = "BDT";
    public string? Token { get; set; }
    public string? Description { get; set; }
    public string? CustomerId { get; set; }
    public string? OrderId { get; set; }
    public Dictionary<string, string> Metadata { get; set; } = new();
}

public class GatewayRefundRequest
{
    public string TransactionId { get; set; } = string.Empty;
    public decimal Amount { get; set; }
    public string Currency { get; set; } = "BDT";
    public string? Reason { get; set; }
}

public class GatewayResponse
{
    public bool Success { get; set; }
    public string TransactionId { get; set; } = string.Empty;
    public string? ErrorCode { get; set; }
    public string? ErrorMessage { get; set; }
    public string? RawResponse { get; set; }
}

public interface IPaymentGateway
{
    string Name { get; }
    Task<GatewayResponse> ChargeAsync(GatewayChargeRequest request, CancellationToken cancellationToken = default);
    Task<GatewayResponse> RefundAsync(GatewayRefundRequest request, CancellationToken cancellationToken = default);
    Task<GatewayResponse> VoidAsync(string transactionId, CancellationToken cancellationToken = default);
    Task<string> TokenizeCardAsync(string cardNumber, int expiryMonth, int expiryYear, string cvv, CancellationToken cancellationToken = default);
}
