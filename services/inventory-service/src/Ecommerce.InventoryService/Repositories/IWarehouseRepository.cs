using Ecommerce.InventoryService.Entities;

namespace Ecommerce.InventoryService.Repositories;

public interface IWarehouseRepository
{
    Task<Warehouse?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<Warehouse?> GetByCodeAsync(string tenantId, string code, CancellationToken cancellationToken = default);
    Task<List<Warehouse>> GetByTenantAsync(string tenantId, CancellationToken cancellationToken = default);
    Task<(List<Warehouse> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default);
    Task<Warehouse> CreateAsync(Warehouse warehouse, CancellationToken cancellationToken = default);
    Task<Warehouse> UpdateAsync(Warehouse warehouse, CancellationToken cancellationToken = default);
    Task DeleteAsync(Guid id, CancellationToken cancellationToken = default);
    Task<bool> CodeExistsAsync(string tenantId, string code, Guid? excludeId = null, CancellationToken cancellationToken = default);
    Task<Warehouse?> GetDefaultWarehouseAsync(string tenantId, CancellationToken cancellationToken = default);
}
