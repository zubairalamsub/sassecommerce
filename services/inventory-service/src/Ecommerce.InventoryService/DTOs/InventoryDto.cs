namespace Ecommerce.InventoryService.DTOs;

public class CreateInventoryItemRequest
{
    public string TenantId { get; set; } = string.Empty;
    public Guid WarehouseId { get; set; }
    public string ProductId { get; set; } = string.Empty;
    public string? VariantId { get; set; }
    public string SKU { get; set; } = string.Empty;
    public int InitialQuantity { get; set; } = 0;
    public int ReorderPoint { get; set; } = 0;
    public int ReorderQuantity { get; set; } = 0;
    public int? MaxStock { get; set; }
    public string? BinLocation { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
}

public class UpdateInventoryItemRequest
{
    public int? ReorderPoint { get; set; }
    public int? ReorderQuantity { get; set; }
    public int? MaxStock { get; set; }
    public string? BinLocation { get; set; }
    public string UpdatedBy { get; set; } = string.Empty;
}

public class AdjustStockRequest
{
    public int Quantity { get; set; }
    public string Reason { get; set; } = string.Empty;
    public string? Notes { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
}

public class TransferStockRequest
{
    public Guid ToWarehouseId { get; set; }
    public int Quantity { get; set; }
    public string? Reason { get; set; }
    public string? Notes { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
}

public class ReserveStockRequest
{
    public string TenantId { get; set; } = string.Empty;
    public string ProductId { get; set; } = string.Empty;
    public string? VariantId { get; set; }
    public int Quantity { get; set; }
    public string OrderId { get; set; } = string.Empty;
    public string OrderItemId { get; set; } = string.Empty;
    public int ExpirationMinutes { get; set; } = 30;
    public string CreatedBy { get; set; } = string.Empty;
}

public class InventoryItemResponse
{
    public Guid Id { get; set; }
    public string TenantId { get; set; } = string.Empty;
    public Guid WarehouseId { get; set; }
    public string WarehouseName { get; set; } = string.Empty;
    public string ProductId { get; set; } = string.Empty;
    public string? VariantId { get; set; }
    public string SKU { get; set; } = string.Empty;
    public int QuantityOnHand { get; set; }
    public int QuantityReserved { get; set; }
    public int QuantityAvailable { get; set; }
    public int ReorderPoint { get; set; }
    public int ReorderQuantity { get; set; }
    public int? MaxStock { get; set; }
    public string? BinLocation { get; set; }
    public DateTime? LastStockCheckAt { get; set; }
    public DateTime? LastReceivedAt { get; set; }
    public bool NeedsReorder { get; set; }
    public DateTime CreatedAt { get; set; }
    public DateTime UpdatedAt { get; set; }
}

public class StockLevelResponse
{
    public string ProductId { get; set; } = string.Empty;
    public string? VariantId { get; set; }
    public string SKU { get; set; } = string.Empty;
    public int TotalOnHand { get; set; }
    public int TotalReserved { get; set; }
    public int TotalAvailable { get; set; }
    public List<WarehouseStockLevel> WarehouseLevels { get; set; } = new();
}

public class WarehouseStockLevel
{
    public Guid WarehouseId { get; set; }
    public string WarehouseName { get; set; } = string.Empty;
    public int QuantityOnHand { get; set; }
    public int QuantityReserved { get; set; }
    public int QuantityAvailable { get; set; }
}

public class StockMovementResponse
{
    public Guid Id { get; set; }
    public string TenantId { get; set; } = string.Empty;
    public Guid InventoryItemId { get; set; }
    public Guid WarehouseId { get; set; }
    public string WarehouseName { get; set; } = string.Empty;
    public string MovementType { get; set; } = string.Empty;
    public int Quantity { get; set; }
    public int QuantityBefore { get; set; }
    public int QuantityAfter { get; set; }
    public string? Reference { get; set; }
    public string? OrderId { get; set; }
    public string? Reason { get; set; }
    public string? Notes { get; set; }
    public DateTime MovementDate { get; set; }
    public Guid? FromWarehouseId { get; set; }
    public Guid? ToWarehouseId { get; set; }
    public string CreatedBy { get; set; } = string.Empty;
    public DateTime CreatedAt { get; set; }
}

public class StockReservationResponse
{
    public Guid Id { get; set; }
    public string TenantId { get; set; } = string.Empty;
    public Guid InventoryItemId { get; set; }
    public string OrderId { get; set; } = string.Empty;
    public string OrderItemId { get; set; } = string.Empty;
    public int QuantityReserved { get; set; }
    public string Status { get; set; } = string.Empty;
    public DateTime ReservedAt { get; set; }
    public DateTime ExpiresAt { get; set; }
    public DateTime? FulfilledAt { get; set; }
    public DateTime? CancelledAt { get; set; }
    public string? CancellationReason { get; set; }
}
