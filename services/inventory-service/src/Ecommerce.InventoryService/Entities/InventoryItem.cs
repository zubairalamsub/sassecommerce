namespace Ecommerce.InventoryService.Entities;

public class InventoryItem : BaseEntity
{
    public string TenantId { get; set; } = string.Empty;
    public Guid WarehouseId { get; set; }
    public string ProductId { get; set; } = string.Empty;
    public string? VariantId { get; set; }
    public string SKU { get; set; } = string.Empty;

    // Stock quantities
    public int QuantityOnHand { get; set; } = 0;
    public int QuantityReserved { get; set; } = 0;
    public int QuantityAvailable => QuantityOnHand - QuantityReserved;

    // Reorder settings
    public int ReorderPoint { get; set; } = 0;
    public int ReorderQuantity { get; set; } = 0;
    public int? MaxStock { get; set; }

    // Location in warehouse
    public string? BinLocation { get; set; }

    // Tracking
    public DateTime? LastStockCheckAt { get; set; }
    public DateTime? LastReceivedAt { get; set; }

    // Navigation properties
    public Warehouse Warehouse { get; set; } = null!;
    public ICollection<StockMovement> StockMovements { get; set; } = new List<StockMovement>();
    public ICollection<StockReservation> StockReservations { get; set; } = new List<StockReservation>();
}
