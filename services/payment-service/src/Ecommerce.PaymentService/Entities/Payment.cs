namespace Ecommerce.PaymentService.Entities;

public enum PaymentStatus
{
    Pending,
    Processing,
    Completed,
    Failed,
    Cancelled,
    Refunded,
    PartiallyRefunded
}

public enum PaymentMethodType
{
    CreditCard,
    DebitCard,
    BankTransfer,
    DigitalWallet,
    bKash,
    Nagad,
    Rocket,
    CashOnDelivery
}

public class Payment : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public string OrderId { get; set; } = string.Empty;
    public string CustomerId { get; set; } = string.Empty;

    // Amount details
    public decimal Amount { get; set; }
    public decimal RefundedAmount { get; set; }
    public string Currency { get; set; } = "BDT";

    // Payment method
    public PaymentMethodType Method { get; set; }
    public Guid? PaymentMethodId { get; set; }

    // Status
    public PaymentStatus Status { get; set; } = PaymentStatus.Pending;
    public string? FailureReason { get; set; }

    // Gateway details
    public string? GatewayName { get; set; }
    public string? GatewayTransactionId { get; set; }
    public string? GatewayResponse { get; set; }

    // Timestamps
    public DateTime? ProcessedAt { get; set; }
    public DateTime? CompletedAt { get; set; }
    public DateTime? FailedAt { get; set; }
    public DateTime? CancelledAt { get; set; }

    // Metadata
    public string? Description { get; set; }
    public string? IdempotencyKey { get; set; }

    // Navigation properties
    public PaymentMethod? PaymentMethodNavigation { get; set; }
    public ICollection<PaymentTransaction> Transactions { get; set; } = new List<PaymentTransaction>();
    public ICollection<Refund> Refunds { get; set; } = new List<Refund>();
}
