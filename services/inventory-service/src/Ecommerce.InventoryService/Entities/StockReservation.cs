namespace Ecommerce.InventoryService.Entities;

public enum ReservationStatus
{
    Active,
    Fulfilled,
    Cancelled,
    Expired
}

public class StockReservation : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public Guid InventoryItemId { get; set; }
    public string OrderId { get; set; } = string.Empty;
    public string OrderItemId { get; set; } = string.Empty;
    public int QuantityReserved { get; set; }
    public ReservationStatus Status { get; set; } = ReservationStatus.Active;
    public DateTime ReservedAt { get; set; }
    public DateTime ExpiresAt { get; set; }
    public DateTime? FulfilledAt { get; set; }
    public DateTime? CancelledAt { get; set; }
    public string? CancellationReason { get; set; }

    // Navigation properties
    public InventoryItem InventoryItem { get; set; } = null!;
}
