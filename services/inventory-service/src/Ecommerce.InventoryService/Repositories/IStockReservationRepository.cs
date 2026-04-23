using Ecommerce.InventoryService.Entities;

namespace Ecommerce.InventoryService.Repositories;

public interface IStockReservationRepository
{
    Task<StockReservation?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<StockReservation>> GetByOrderAsync(string orderId, CancellationToken cancellationToken = default);
    Task<List<StockReservation>> GetActiveReservationsAsync(Guid inventoryItemId, CancellationToken cancellationToken = default);
    Task<List<StockReservation>> GetExpiredReservationsAsync(CancellationToken cancellationToken = default);
    Task<StockReservation> CreateAsync(StockReservation reservation, CancellationToken cancellationToken = default);
    Task<StockReservation> UpdateAsync(StockReservation reservation, CancellationToken cancellationToken = default);
    Task<int> GetTotalReservedQuantityAsync(Guid inventoryItemId, CancellationToken cancellationToken = default);
}
