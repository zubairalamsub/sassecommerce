using Ecommerce.InventoryService.Entities;

namespace Ecommerce.InventoryService.Repositories;

public interface IInventoryRepository
{
    Task<InventoryItem?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<InventoryItem?> GetByProductAsync(string tenantId, Guid warehouseId, string productId, string? variantId, CancellationToken cancellationToken = default);
    Task<List<InventoryItem>> GetByProductIdAsync(string tenantId, string productId, CancellationToken cancellationToken = default);
    Task<List<InventoryItem>> GetByWarehouseAsync(Guid warehouseId, CancellationToken cancellationToken = default);
    Task<List<InventoryItem>> GetLowStockItemsAsync(string tenantId, CancellationToken cancellationToken = default);
    Task<(List<InventoryItem> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default);
    Task<InventoryItem> CreateAsync(InventoryItem inventoryItem, CancellationToken cancellationToken = default);
    Task<InventoryItem> UpdateAsync(InventoryItem inventoryItem, CancellationToken cancellationToken = default);
    Task DeleteAsync(Guid id, CancellationToken cancellationToken = default);
    Task<bool> ProductExistsInWarehouseAsync(string tenantId, Guid warehouseId, string productId, string? variantId, Guid? excludeId = null, CancellationToken cancellationToken = default);
}
