using Ecommerce.InventoryService.Entities;

namespace Ecommerce.InventoryService.Repositories;

public interface IStockMovementRepository
{
    Task<StockMovement?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<StockMovement>> GetByInventoryItemAsync(Guid inventoryItemId, CancellationToken cancellationToken = default);
    Task<List<StockMovement>> GetByOrderAsync(string orderId, CancellationToken cancellationToken = default);
    Task<(List<StockMovement> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, DateTime? startDate = null, DateTime? endDate = null, CancellationToken cancellationToken = default);
    Task<StockMovement> CreateAsync(StockMovement stockMovement, CancellationToken cancellationToken = default);
}
