namespace Ecommerce.PaymentService.Entities;

public class PaymentMethod : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public string CustomerId { get; set; } = string.Empty;

    // Payment method type
    public PaymentMethodType Type { get; set; }

    // Tokenized card details (no raw card data stored)
    public string? Token { get; set; }
    public string? Last4 { get; set; }
    public string? Brand { get; set; }
    public int? ExpiryMonth { get; set; }
    public int? ExpiryYear { get; set; }
    public string? CardholderName { get; set; }

    // Bank account details (tokenized)
    public string? BankName { get; set; }
    public string? AccountLast4 { get; set; }

    // Digital wallet
    public string? WalletProvider { get; set; }
    public string? WalletEmail { get; set; }

    // Status
    public bool IsDefault { get; set; }
    public bool IsActive { get; set; } = true;

    // Billing address
    public string? BillingAddressLine1 { get; set; }
    public string? BillingAddressLine2 { get; set; }
    public string? BillingCity { get; set; }
    public string? BillingState { get; set; }
    public string? BillingPostalCode { get; set; }
    public string? BillingCountry { get; set; }

    // Navigation
    public ICollection<Payment> Payments { get; set; } = new List<Payment>();
}
