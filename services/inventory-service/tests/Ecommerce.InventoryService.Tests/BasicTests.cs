using Ecommerce.InventoryService.Data;
using Ecommerce.InventoryService.Entities;
using FluentAssertions;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.InventoryService.Tests;

public class BasicTests : IDisposable
{
    private readonly InventoryDbContext _context;

    public BasicTests()
    {
        var options = new DbContextOptionsBuilder<InventoryDbContext>()
            .UseInMemoryDatabase(databaseName: Guid.NewGuid().ToString())
            .Options;

        _context = new InventoryDbContext(options);
    }

    [Fact]
    public async Task CanCreateWarehouse()
    {
        // Arrange
        var warehouse = new Warehouse
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            Code = "WH001",
            Name = "Main Warehouse",
            Address = "123 Main St",
            City = "New York",
            State = "NY",
            Country = "BD",
            PostalCode = "10001",
            IsActive = true,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        // Act
        _context.Warehouses.Add(warehouse);
        await _context.SaveChangesAsync();

        // Assert
        var result = await _context.Warehouses.FindAsync(warehouse.Id);
        result.Should().NotBeNull();
        result!.Code.Should().Be("WH001");
        result.Name.Should().Be("Main Warehouse");
    }

    [Fact]
    public async Task CanCreateInventoryItem()
    {
        // Arrange
        var warehouse = new Warehouse
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            Code = "WH001",
            Name = "Test Warehouse",
            Address = "123 Main St",
            City = "New York",
            State = "NY",
            Country = "BD",
            PostalCode = "10001",
            IsActive = true,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        _context.Warehouses.Add(warehouse);
        await _context.SaveChangesAsync();

        var item = new InventoryItem
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            WarehouseId = warehouse.Id,
            ProductId = "prod-001",
            SKU = "SKU-001",
            QuantityOnHand = 100,
            QuantityReserved = 20,
            ReorderPoint = 50,
            ReorderQuantity = 100,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        // Act
        _context.InventoryItems.Add(item);
        await _context.SaveChangesAsync();

        // Assert
        var result = await _context.InventoryItems.FindAsync(item.Id);
        result.Should().NotBeNull();
        result!.ProductId.Should().Be("prod-001");
        result.QuantityOnHand.Should().Be(100);
        result.QuantityAvailable.Should().Be(80); // 100 - 20
    }

    [Fact]
    public async Task CanCreateStockMovement()
    {
        // Arrange
        var warehouse = new Warehouse
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            Code = "WH001",
            Name = "Test Warehouse",
            Address = "123 Main St",
            City = "New York",
            State = "NY",
            Country = "BD",
            PostalCode = "10001",
            IsActive = true,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        _context.Warehouses.Add(warehouse);

        var item = new InventoryItem
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            WarehouseId = warehouse.Id,
            ProductId = "prod-001",
            SKU = "SKU-001",
            QuantityOnHand = 100,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        _context.InventoryItems.Add(item);
        await _context.SaveChangesAsync();

        var movement = new StockMovement
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            WarehouseId = warehouse.Id,
            InventoryItemId = item.Id,
            MovementType = MovementType.Adjustment,
            Quantity = 50,
            QuantityBefore = 100,
            QuantityAfter = 150,
            Reason = "Stock receipt",
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        // Act
        _context.StockMovements.Add(movement);
        await _context.SaveChangesAsync();

        // Assert
        var result = await _context.StockMovements.FindAsync(movement.Id);
        result.Should().NotBeNull();
        result!.MovementType.Should().Be(MovementType.Adjustment);
        result.Quantity.Should().Be(50);
    }

    [Fact]
    public async Task CanCreateStockReservation()
    {
        // Arrange
        var warehouse = new Warehouse
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            Code = "WH001",
            Name = "Test Warehouse",
            Address = "123 Main St",
            City = "New York",
            State = "NY",
            Country = "BD",
            PostalCode = "10001",
            IsActive = true,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        _context.Warehouses.Add(warehouse);

        var item = new InventoryItem
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            WarehouseId = warehouse.Id,
            ProductId = "prod-001",
            SKU = "SKU-001",
            QuantityOnHand = 100,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        _context.InventoryItems.Add(item);
        await _context.SaveChangesAsync();

        var reservation = new StockReservation
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            InventoryItemId = item.Id,
            OrderId = "order-123",
            OrderItemId = "item-456",
            QuantityReserved = 10,
            Status = ReservationStatus.Active,
            ReservedAt = DateTime.UtcNow,
            ExpiresAt = DateTime.UtcNow.AddMinutes(30),
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        // Act
        _context.StockReservations.Add(reservation);
        await _context.SaveChangesAsync();

        // Assert
        var result = await _context.StockReservations.FindAsync(reservation.Id);
        result.Should().NotBeNull();
        result!.Status.Should().Be(ReservationStatus.Active);
        result.QuantityReserved.Should().Be(10);
    }

    [Fact]
    public async Task SoftDelete_FiltersDeletedEntities()
    {
        // Arrange
        var warehouse = new Warehouse
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant1",
            Code = "WH001",
            Name = "To Be Deleted",
            Address = "123 Main St",
            City = "New York",
            State = "NY",
            Country = "BD",
            PostalCode = "10001",
            IsActive = true,
            CreatedBy = "test",
            UpdatedBy = "test"
        };

        _context.Warehouses.Add(warehouse);
        await _context.SaveChangesAsync();

        // Act
        warehouse.DeletedAt = DateTime.UtcNow;
        await _context.SaveChangesAsync();

        // Assert - FindAsync doesn't respect query filters, so use FirstOrDefaultAsync
        var result = await _context.Warehouses.FirstOrDefaultAsync(w => w.Id == warehouse.Id);
        result.Should().BeNull(); // Soft delete query filter should exclude it

        var deletedWarehouse = await _context.Warehouses.IgnoreQueryFilters().FirstOrDefaultAsync(w => w.Id == warehouse.Id);
        deletedWarehouse.Should().NotBeNull();
        deletedWarehouse!.DeletedAt.Should().NotBeNull();
    }

    public void Dispose()
    {
        _context.Database.EnsureDeleted();
        _context.Dispose();
    }
}
