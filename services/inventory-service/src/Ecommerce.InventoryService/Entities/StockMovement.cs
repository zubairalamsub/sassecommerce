namespace Ecommerce.InventoryService.Entities;

public enum MovementType
{
    Receipt,        // Stock received
    Shipment,       // Stock shipped out
    Adjustment,     // Manual adjustment
    Transfer,       // Transfer between warehouses
    Return,         // Stock returned
    Damage,         // Damaged stock
    Loss,           // Lost stock
    Reservation,    // Reserved for order
    Release         // Reservation released
}

public class StockMovement : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public Guid InventoryItemId { get; set; }
    public Guid WarehouseId { get; set; }
    public MovementType MovementType { get; set; }
    public int Quantity { get; set; }
    public int QuantityBefore { get; set; }
    public int QuantityAfter { get; set; }
    public string? Reference { get; set; }
    public string? OrderId { get; set; }
    public string? Reason { get; set; }
    public string? Notes { get; set; }
    public DateTime MovementDate { get; set; }

    // For transfers
    public Guid? FromWarehouseId { get; set; }
    public Guid? ToWarehouseId { get; set; }

    // Navigation properties
    public InventoryItem InventoryItem { get; set; } = null!;
    public Warehouse Warehouse { get; set; } = null!;
}
