namespace Ecommerce.PaymentService.Entities;

public enum RefundStatus
{
    Pending,
    Processing,
    Completed,
    Failed
}

public class Refund : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public Guid PaymentId { get; set; }

    // Refund details
    public decimal Amount { get; set; }
    public string Currency { get; set; } = "BDT";
    public string Reason { get; set; } = string.Empty;

    // Status
    public RefundStatus Status { get; set; } = RefundStatus.Pending;
    public string? FailureReason { get; set; }

    // Gateway details
    public string? GatewayRefundId { get; set; }
    public string? GatewayResponse { get; set; }

    // Timestamps
    public DateTime? ProcessedAt { get; set; }
    public DateTime? CompletedAt { get; set; }
    public DateTime? FailedAt { get; set; }

    // Navigation
    public Payment Payment { get; set; } = null!;
}
