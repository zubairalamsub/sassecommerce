using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.InventoryService.Repositories;

public class StockReservationRepository : IStockReservationRepository
{
    private readonly InventoryDbContext _context;

    public StockReservationRepository(InventoryDbContext context)
    {
        _context = context;
    }

    public async Task<StockReservation?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.StockReservations
            .Include(sr => sr.InventoryItem)
            .FirstOrDefaultAsync(sr => sr.Id == id, cancellationToken);
    }

    public async Task<List<StockReservation>> GetByOrderAsync(string orderId, CancellationToken cancellationToken = default)
    {
        return await _context.StockReservations
            .Include(sr => sr.InventoryItem)
            .Where(sr => sr.OrderId == orderId)
            .ToListAsync(cancellationToken);
    }

    public async Task<List<StockReservation>> GetActiveReservationsAsync(Guid inventoryItemId, CancellationToken cancellationToken = default)
    {
        return await _context.StockReservations
            .Where(sr => sr.InventoryItemId == inventoryItemId &&
                        sr.Status == ReservationStatus.Active &&
                        sr.ExpiresAt > DateTime.UtcNow)
            .ToListAsync(cancellationToken);
    }

    public async Task<List<StockReservation>> GetExpiredReservationsAsync(CancellationToken cancellationToken = default)
    {
        return await _context.StockReservations
            .Include(sr => sr.InventoryItem)
            .Where(sr => sr.Status == ReservationStatus.Active &&
                        sr.ExpiresAt <= DateTime.UtcNow)
            .ToListAsync(cancellationToken);
    }

    public async Task<StockReservation> CreateAsync(StockReservation reservation, CancellationToken cancellationToken = default)
    {
        _context.StockReservations.Add(reservation);
        await _context.SaveChangesAsync(cancellationToken);
        return reservation;
    }

    public async Task<StockReservation> UpdateAsync(StockReservation reservation, CancellationToken cancellationToken = default)
    {
        _context.StockReservations.Update(reservation);
        await _context.SaveChangesAsync(cancellationToken);
        return reservation;
    }

    public async Task<int> GetTotalReservedQuantityAsync(Guid inventoryItemId, CancellationToken cancellationToken = default)
    {
        return await _context.StockReservations
            .Where(sr => sr.InventoryItemId == inventoryItemId &&
                        sr.Status == ReservationStatus.Active &&
                        sr.ExpiresAt > DateTime.UtcNow)
            .SumAsync(sr => sr.QuantityReserved, cancellationToken);
    }
}
