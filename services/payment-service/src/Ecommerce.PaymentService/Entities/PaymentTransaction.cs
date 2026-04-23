namespace Ecommerce.PaymentService.Entities;

public enum TransactionType
{
    Authorization,
    Capture,
    Charge,
    Refund,
    Void,
    Chargeback
}

public enum TransactionStatus
{
    Pending,
    Success,
    Failed,
    Reversed
}

public class PaymentTransaction : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public Guid PaymentId { get; set; }

    // Transaction details
    public TransactionType Type { get; set; }
    public TransactionStatus Status { get; set; } = TransactionStatus.Pending;
    public decimal Amount { get; set; }
    public string Currency { get; set; } = "BDT";

    // Gateway details
    public string? GatewayTransactionId { get; set; }
    public string? GatewayResponse { get; set; }
    public string? GatewayErrorCode { get; set; }
    public string? GatewayErrorMessage { get; set; }

    // Metadata
    public string? Reference { get; set; }
    public string? Notes { get; set; }
    public DateTime TransactionDate { get; set; }

    // Navigation
    public Payment Payment { get; set; } = null!;
}
