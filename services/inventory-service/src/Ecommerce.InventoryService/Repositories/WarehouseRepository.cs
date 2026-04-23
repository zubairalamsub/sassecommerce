using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.InventoryService.Repositories;

public class WarehouseRepository : IWarehouseRepository
{
    private readonly InventoryDbContext _context;

    public WarehouseRepository(InventoryDbContext context)
    {
        _context = context;
    }

    public async Task<Warehouse?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.Warehouses
            .FirstOrDefaultAsync(w => w.Id == id, cancellationToken);
    }

    public async Task<Warehouse?> GetByCodeAsync(string tenantId, string code, CancellationToken cancellationToken = default)
    {
        return await _context.Warehouses
            .FirstOrDefaultAsync(w => w.TenantId == tenantId && w.Code == code, cancellationToken);
    }

    public async Task<List<Warehouse>> GetByTenantAsync(string tenantId, CancellationToken cancellationToken = default)
    {
        return await _context.Warehouses
            .Where(w => w.TenantId == tenantId)
            .OrderBy(w => w.Name)
            .ToListAsync(cancellationToken);
    }

    public async Task<(List<Warehouse> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default)
    {
        var query = _context.Warehouses.Where(w => w.TenantId == tenantId);

        var total = await query.CountAsync(cancellationToken);
        var items = await query
            .OrderBy(w => w.Name)
            .Skip(offset)
            .Take(limit)
            .ToListAsync(cancellationToken);

        return (items, total);
    }

    public async Task<Warehouse> CreateAsync(Warehouse warehouse, CancellationToken cancellationToken = default)
    {
        _context.Warehouses.Add(warehouse);
        await _context.SaveChangesAsync(cancellationToken);
        return warehouse;
    }

    public async Task<Warehouse> UpdateAsync(Warehouse warehouse, CancellationToken cancellationToken = default)
    {
        _context.Warehouses.Update(warehouse);
        await _context.SaveChangesAsync(cancellationToken);
        return warehouse;
    }

    public async Task DeleteAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var warehouse = await GetByIdAsync(id, cancellationToken);
        if (warehouse != null)
        {
            warehouse.DeletedAt = DateTime.UtcNow;
            await _context.SaveChangesAsync(cancellationToken);
        }
    }

    public async Task<bool> CodeExistsAsync(string tenantId, string code, Guid? excludeId = null, CancellationToken cancellationToken = default)
    {
        var query = _context.Warehouses
            .Where(w => w.TenantId == tenantId && w.Code == code);

        if (excludeId.HasValue)
        {
            query = query.Where(w => w.Id != excludeId.Value);
        }

        return await query.AnyAsync(cancellationToken);
    }

    public async Task<Warehouse?> GetDefaultWarehouseAsync(string tenantId, CancellationToken cancellationToken = default)
    {
        return await _context.Warehouses
            .FirstOrDefaultAsync(w => w.TenantId == tenantId && w.IsDefault && w.IsActive, cancellationToken);
    }
}
