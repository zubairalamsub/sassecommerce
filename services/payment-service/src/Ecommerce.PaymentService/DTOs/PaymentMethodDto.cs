namespace Ecommerce.PaymentService.DTOs;

// === Payment Method Request DTOs ===

public class CreatePaymentMethodRequest
{
    public string TenantId { get; set; } = string.Empty;
    public string CustomerId { get; set; } = string.Empty;
    public string Type { get; set; } = "credit_card";

    // Card details (will be tokenized, not stored raw)
    public string? CardNumber { get; set; }
    public int? ExpiryMonth { get; set; }
    public int? ExpiryYear { get; set; }
    public string? CardholderName { get; set; }

    // Bank account
    public string? BankName { get; set; }
    public string? AccountNumber { get; set; }
    public string? RoutingNumber { get; set; }

    // Digital wallet
    public string? WalletProvider { get; set; }
    public string? WalletEmail { get; set; }

    // Billing address
    public string? BillingAddressLine1 { get; set; }
    public string? BillingAddressLine2 { get; set; }
    public string? BillingCity { get; set; }
    public string? BillingState { get; set; }
    public string? BillingPostalCode { get; set; }
    public string? BillingCountry { get; set; }

    public bool IsDefault { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
}

public class UpdatePaymentMethodRequest
{
    public int? ExpiryMonth { get; set; }
    public int? ExpiryYear { get; set; }
    public string? CardholderName { get; set; }
    public string? BillingAddressLine1 { get; set; }
    public string? BillingAddressLine2 { get; set; }
    public string? BillingCity { get; set; }
    public string? BillingState { get; set; }
    public string? BillingPostalCode { get; set; }
    public string? BillingCountry { get; set; }
    public bool? IsDefault { get; set; }
    public string UpdatedBy { get; set; } = string.Empty;
}

// === Payment Method Response DTO ===

public class PaymentMethodResponse
{
    public Guid Id { get; set; }
    public string TenantId { get; set; } = string.Empty;
    public string CustomerId { get; set; } = string.Empty;
    public string Type { get; set; } = string.Empty;

    // Masked card info
    public string? Last4 { get; set; }
    public string? Brand { get; set; }
    public int? ExpiryMonth { get; set; }
    public int? ExpiryYear { get; set; }
    public string? CardholderName { get; set; }

    // Bank info
    public string? BankName { get; set; }
    public string? AccountLast4 { get; set; }

    // Wallet info
    public string? WalletProvider { get; set; }
    public string? WalletEmail { get; set; }

    // Status
    public bool IsDefault { get; set; }
    public bool IsActive { get; set; }

    // Billing address
    public string? BillingAddressLine1 { get; set; }
    public string? BillingCity { get; set; }
    public string? BillingState { get; set; }
    public string? BillingPostalCode { get; set; }
    public string? BillingCountry { get; set; }

    public DateTime CreatedAt { get; set; }
    public DateTime UpdatedAt { get; set; }
}
