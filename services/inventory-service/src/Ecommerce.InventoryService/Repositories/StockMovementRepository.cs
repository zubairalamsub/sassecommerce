using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.InventoryService.Repositories;

public class StockMovementRepository : IStockMovementRepository
{
    private readonly InventoryDbContext _context;

    public StockMovementRepository(InventoryDbContext context)
    {
        _context = context;
    }

    public async Task<StockMovement?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.StockMovements
            .Include(sm => sm.InventoryItem)
            .Include(sm => sm.Warehouse)
            .FirstOrDefaultAsync(sm => sm.Id == id, cancellationToken);
    }

    public async Task<List<StockMovement>> GetByInventoryItemAsync(Guid inventoryItemId, CancellationToken cancellationToken = default)
    {
        return await _context.StockMovements
            .Include(sm => sm.Warehouse)
            .Where(sm => sm.InventoryItemId == inventoryItemId)
            .OrderByDescending(sm => sm.MovementDate)
            .ToListAsync(cancellationToken);
    }

    public async Task<List<StockMovement>> GetByOrderAsync(string orderId, CancellationToken cancellationToken = default)
    {
        return await _context.StockMovements
            .Include(sm => sm.InventoryItem)
            .Include(sm => sm.Warehouse)
            .Where(sm => sm.OrderId == orderId)
            .OrderBy(sm => sm.MovementDate)
            .ToListAsync(cancellationToken);
    }

    public async Task<(List<StockMovement> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, DateTime? startDate = null, DateTime? endDate = null, CancellationToken cancellationToken = default)
    {
        var query = _context.StockMovements
            .Include(sm => sm.InventoryItem)
            .Include(sm => sm.Warehouse)
            .Where(sm => sm.TenantId == tenantId);

        if (startDate.HasValue)
        {
            query = query.Where(sm => sm.MovementDate >= startDate.Value);
        }

        if (endDate.HasValue)
        {
            query = query.Where(sm => sm.MovementDate <= endDate.Value);
        }

        var total = await query.CountAsync(cancellationToken);
        var items = await query
            .OrderByDescending(sm => sm.MovementDate)
            .Skip(offset)
            .Take(limit)
            .ToListAsync(cancellationToken);

        return (items, total);
    }

    public async Task<StockMovement> CreateAsync(StockMovement stockMovement, CancellationToken cancellationToken = default)
    {
        _context.StockMovements.Add(stockMovement);
        await _context.SaveChangesAsync(cancellationToken);
        return stockMovement;
    }
}
