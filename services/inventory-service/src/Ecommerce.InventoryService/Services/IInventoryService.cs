using Ecommerce.InventoryService.DTOs;

namespace Ecommerce.InventoryService.Services;

public interface IInventoryService
{
    // Warehouse operations
    Task<WarehouseResponse> CreateWarehouseAsync(CreateWarehouseRequest request, CancellationToken cancellationToken = default);
    Task<WarehouseResponse> UpdateWarehouseAsync(Guid id, UpdateWarehouseRequest request, CancellationToken cancellationToken = default);
    Task<WarehouseResponse?> GetWarehouseByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<WarehouseResponse>> GetWarehousesByTenantAsync(string tenantId, CancellationToken cancellationToken = default);
    Task<(List<WarehouseResponse> Items, int Total)> GetWarehousesPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default);
    Task DeleteWarehouseAsync(Guid id, CancellationToken cancellationToken = default);

    // Inventory item operations
    Task<InventoryItemResponse> CreateInventoryItemAsync(CreateInventoryItemRequest request, CancellationToken cancellationToken = default);
    Task<InventoryItemResponse> UpdateInventoryItemAsync(Guid id, UpdateInventoryItemRequest request, CancellationToken cancellationToken = default);
    Task<InventoryItemResponse?> GetInventoryItemByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<StockLevelResponse?> GetStockLevelAsync(string tenantId, string productId, string? variantId = null, CancellationToken cancellationToken = default);
    Task<List<InventoryItemResponse>> GetLowStockItemsAsync(string tenantId, CancellationToken cancellationToken = default);
    Task<(List<InventoryItemResponse> Items, int Total)> GetInventoryItemsPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default);
    Task DeleteInventoryItemAsync(Guid id, CancellationToken cancellationToken = default);

    // Stock operations
    Task<InventoryItemResponse> AdjustStockAsync(Guid inventoryItemId, AdjustStockRequest request, CancellationToken cancellationToken = default);
    Task<InventoryItemResponse> TransferStockAsync(Guid inventoryItemId, TransferStockRequest request, CancellationToken cancellationToken = default);
    Task<StockReservationResponse> ReserveStockAsync(ReserveStockRequest request, CancellationToken cancellationToken = default);
    Task<StockReservationResponse> FulfillReservationAsync(Guid reservationId, string fulfilledBy, CancellationToken cancellationToken = default);
    Task<StockReservationResponse> CancelReservationAsync(Guid reservationId, string cancelledBy, string? reason = null, CancellationToken cancellationToken = default);
    Task ProcessExpiredReservationsAsync(CancellationToken cancellationToken = default);

    // Stock movement history
    Task<(List<StockMovementResponse> Items, int Total)> GetStockMovementsPagedAsync(string tenantId, int offset, int limit, DateTime? startDate = null, DateTime? endDate = null, CancellationToken cancellationToken = default);
    Task<List<StockMovementResponse>> GetStockMovementsByOrderAsync(string orderId, CancellationToken cancellationToken = default);
}
