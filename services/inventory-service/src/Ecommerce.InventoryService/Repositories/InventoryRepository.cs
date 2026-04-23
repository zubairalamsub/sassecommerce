using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.InventoryService.Repositories;

public class InventoryRepository : IInventoryRepository
{
    private readonly InventoryDbContext _context;

    public InventoryRepository(InventoryDbContext context)
    {
        _context = context;
    }

    public async Task<InventoryItem?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.InventoryItems
            .Include(i => i.Warehouse)
            .FirstOrDefaultAsync(i => i.Id == id, cancellationToken);
    }

    public async Task<InventoryItem?> GetByProductAsync(string tenantId, Guid warehouseId, string productId, string? variantId, CancellationToken cancellationToken = default)
    {
        return await _context.InventoryItems
            .Include(i => i.Warehouse)
            .FirstOrDefaultAsync(i =>
                i.TenantId == tenantId &&
                i.WarehouseId == warehouseId &&
                i.ProductId == productId &&
                i.VariantId == variantId,
                cancellationToken);
    }

    public async Task<List<InventoryItem>> GetByProductIdAsync(string tenantId, string productId, CancellationToken cancellationToken = default)
    {
        return await _context.InventoryItems
            .Include(i => i.Warehouse)
            .Where(i => i.TenantId == tenantId && i.ProductId == productId)
            .ToListAsync(cancellationToken);
    }

    public async Task<List<InventoryItem>> GetByWarehouseAsync(Guid warehouseId, CancellationToken cancellationToken = default)
    {
        return await _context.InventoryItems
            .Include(i => i.Warehouse)
            .Where(i => i.WarehouseId == warehouseId)
            .OrderBy(i => i.SKU)
            .ToListAsync(cancellationToken);
    }

    public async Task<List<InventoryItem>> GetLowStockItemsAsync(string tenantId, CancellationToken cancellationToken = default)
    {
        return await _context.InventoryItems
            .Include(i => i.Warehouse)
            .Where(i => i.TenantId == tenantId &&
                       (i.QuantityOnHand - i.QuantityReserved) <= i.ReorderPoint)
            .ToListAsync(cancellationToken);
    }

    public async Task<(List<InventoryItem> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default)
    {
        var query = _context.InventoryItems
            .Include(i => i.Warehouse)
            .Where(i => i.TenantId == tenantId);

        var total = await query.CountAsync(cancellationToken);
        var items = await query
            .OrderBy(i => i.SKU)
            .Skip(offset)
            .Take(limit)
            .ToListAsync(cancellationToken);

        return (items, total);
    }

    public async Task<InventoryItem> CreateAsync(InventoryItem inventoryItem, CancellationToken cancellationToken = default)
    {
        _context.InventoryItems.Add(inventoryItem);
        await _context.SaveChangesAsync(cancellationToken);
        return inventoryItem;
    }

    public async Task<InventoryItem> UpdateAsync(InventoryItem inventoryItem, CancellationToken cancellationToken = default)
    {
        _context.InventoryItems.Update(inventoryItem);
        await _context.SaveChangesAsync(cancellationToken);
        return inventoryItem;
    }

    public async Task DeleteAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var item = await GetByIdAsync(id, cancellationToken);
        if (item != null)
        {
            item.DeletedAt = DateTime.UtcNow;
            await _context.SaveChangesAsync(cancellationToken);
        }
    }

    public async Task<bool> ProductExistsInWarehouseAsync(string tenantId, Guid warehouseId, string productId, string? variantId, Guid? excludeId = null, CancellationToken cancellationToken = default)
    {
        var query = _context.InventoryItems
            .Where(i => i.TenantId == tenantId &&
                       i.WarehouseId == warehouseId &&
                       i.ProductId == productId &&
                       i.VariantId == variantId);

        if (excludeId.HasValue)
        {
            query = query.Where(i => i.Id != excludeId.Value);
        }

        return await query.AnyAsync(cancellationToken);
    }
}
