namespace Ecommerce.PaymentService.DTOs;

// === Payment Request DTOs ===

public class CreatePaymentRequest
{
    public string TenantId { get; set; } = string.Empty;
    public string CustomerId { get; set; } = string.Empty;
    public string OrderId { get; set; } = string.Empty;
    public decimal Amount { get; set; }
    public string Currency { get; set; } = "BDT";
    public string Method { get; set; } = "bkash";
    public Guid? PaymentMethodId { get; set; }
    public string? Description { get; set; }
    public string? IdempotencyKey { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
}

public class RefundPaymentRequest
{
    public string Reason { get; set; } = string.Empty;
    public decimal? Amount { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
}

public class CancelPaymentRequest
{
    public string Reason { get; set; } = string.Empty;
    public string CancelledBy { get; set; } = string.Empty;
}

// === Payment Response DTOs ===

public class PaymentResponse
{
    public Guid Id { get; set; }
    public string TenantId { get; set; } = string.Empty;
    public string OrderId { get; set; } = string.Empty;
    public string CustomerId { get; set; } = string.Empty;
    public decimal Amount { get; set; }
    public decimal RefundedAmount { get; set; }
    public string Currency { get; set; } = string.Empty;
    public string Method { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public string? FailureReason { get; set; }
    public string? GatewayName { get; set; }
    public string? GatewayTransactionId { get; set; }
    public string? Description { get; set; }
    public DateTime? ProcessedAt { get; set; }
    public DateTime? CompletedAt { get; set; }
    public DateTime CreatedAt { get; set; }
    public DateTime UpdatedAt { get; set; }
}

public class PaymentDetailResponse : PaymentResponse
{
    public List<PaymentTransactionResponse> Transactions { get; set; } = new();
    public List<RefundResponse> Refunds { get; set; } = new();
}

// === Transaction Response DTO ===

public class PaymentTransactionResponse
{
    public Guid Id { get; set; }
    public Guid PaymentId { get; set; }
    public string Type { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public decimal Amount { get; set; }
    public string Currency { get; set; } = string.Empty;
    public string? GatewayTransactionId { get; set; }
    public string? Reference { get; set; }
    public string? Notes { get; set; }
    public DateTime TransactionDate { get; set; }
    public DateTime CreatedAt { get; set; }
}

// === Refund Response DTO ===

public class RefundResponse
{
    public Guid Id { get; set; }
    public Guid PaymentId { get; set; }
    public decimal Amount { get; set; }
    public string Currency { get; set; } = string.Empty;
    public string Reason { get; set; } = string.Empty;
    public string Status { get; set; } = string.Empty;
    public string? FailureReason { get; set; }
    public string? GatewayRefundId { get; set; }
    public DateTime? ProcessedAt { get; set; }
    public DateTime? CompletedAt { get; set; }
    public DateTime CreatedAt { get; set; }
}
