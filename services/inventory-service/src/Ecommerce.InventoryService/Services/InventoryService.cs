using AutoMapper;
using Ecommerce.InventoryService.DTOs;
using Ecommerce.InventoryService.Entities;
using Ecommerce.InventoryService.Repositories;
using Microsoft.Extensions.Logging;

namespace Ecommerce.InventoryService.Services;

public class InventoryService : IInventoryService
{
    private readonly IWarehouseRepository _warehouseRepository;
    private readonly IInventoryRepository _inventoryRepository;
    private readonly IStockMovementRepository _stockMovementRepository;
    private readonly IStockReservationRepository _stockReservationRepository;
    private readonly IMapper _mapper;
    private readonly ILogger<InventoryService> _logger;

    public InventoryService(
        IWarehouseRepository warehouseRepository,
        IInventoryRepository inventoryRepository,
        IStockMovementRepository stockMovementRepository,
        IStockReservationRepository stockReservationRepository,
        IMapper mapper,
        ILogger<InventoryService> logger)
    {
        _warehouseRepository = warehouseRepository;
        _inventoryRepository = inventoryRepository;
        _stockMovementRepository = stockMovementRepository;
        _stockReservationRepository = stockReservationRepository;
        _mapper = mapper;
        _logger = logger;
    }

    #region Warehouse Operations

    public async Task<WarehouseResponse> CreateWarehouseAsync(CreateWarehouseRequest request, CancellationToken cancellationToken = default)
    {
        if (await _warehouseRepository.CodeExistsAsync(request.TenantId, request.Code, null, cancellationToken))
        {
            throw new InvalidOperationException($"Warehouse with code '{request.Code}' already exists");
        }

        var warehouse = _mapper.Map<Warehouse>(request);
        warehouse.Id = Guid.NewGuid();

        var created = await _warehouseRepository.CreateAsync(warehouse, cancellationToken);
        _logger.LogInformation("Warehouse created: {WarehouseId} ({Code})", created.Id, created.Code);

        return _mapper.Map<WarehouseResponse>(created);
    }

    public async Task<WarehouseResponse> UpdateWarehouseAsync(Guid id, UpdateWarehouseRequest request, CancellationToken cancellationToken = default)
    {
        var warehouse = await _warehouseRepository.GetByIdAsync(id, cancellationToken)
            ?? throw new KeyNotFoundException($"Warehouse {id} not found");

        if (request.Name != null) warehouse.Name = request.Name;
        if (request.Description != null) warehouse.Description = request.Description;
        if (request.Address != null) warehouse.Address = request.Address;
        if (request.City != null) warehouse.City = request.City;
        if (request.State != null) warehouse.State = request.State;
        if (request.Country != null) warehouse.Country = request.Country;
        if (request.PostalCode != null) warehouse.PostalCode = request.PostalCode;
        if (request.Phone != null) warehouse.Phone = request.Phone;
        if (request.Email != null) warehouse.Email = request.Email;
        if (request.IsActive.HasValue) warehouse.IsActive = request.IsActive.Value;
        if (request.IsDefault.HasValue) warehouse.IsDefault = request.IsDefault.Value;
        warehouse.UpdatedBy = request.UpdatedBy;

        var updated = await _warehouseRepository.UpdateAsync(warehouse, cancellationToken);
        _logger.LogInformation("Warehouse updated: {WarehouseId}", id);

        return _mapper.Map<WarehouseResponse>(updated);
    }

    public async Task<WarehouseResponse?> GetWarehouseByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var warehouse = await _warehouseRepository.GetByIdAsync(id, cancellationToken);
        return warehouse == null ? null : _mapper.Map<WarehouseResponse>(warehouse);
    }

    public async Task<List<WarehouseResponse>> GetWarehousesByTenantAsync(string tenantId, CancellationToken cancellationToken = default)
    {
        var warehouses = await _warehouseRepository.GetByTenantAsync(tenantId, cancellationToken);
        return _mapper.Map<List<WarehouseResponse>>(warehouses);
    }

    public async Task<(List<WarehouseResponse> Items, int Total)> GetWarehousesPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default)
    {
        var (warehouses, total) = await _warehouseRepository.GetPagedAsync(tenantId, offset, limit, cancellationToken);
        return (_mapper.Map<List<WarehouseResponse>>(warehouses), total);
    }

    public async Task DeleteWarehouseAsync(Guid id, CancellationToken cancellationToken = default)
    {
        await _warehouseRepository.DeleteAsync(id, cancellationToken);
        _logger.LogInformation("Warehouse deleted: {WarehouseId}", id);
    }

    #endregion

    #region Inventory Item Operations

    public async Task<InventoryItemResponse> CreateInventoryItemAsync(CreateInventoryItemRequest request, CancellationToken cancellationToken = default)
    {
        if (await _inventoryRepository.ProductExistsInWarehouseAsync(
            request.TenantId, request.WarehouseId, request.ProductId, request.VariantId, null, cancellationToken))
        {
            throw new InvalidOperationException("Product already exists in this warehouse");
        }

        var warehouse = await _warehouseRepository.GetByIdAsync(request.WarehouseId, cancellationToken)
            ?? throw new KeyNotFoundException($"Warehouse {request.WarehouseId} not found");

        var inventoryItem = _mapper.Map<InventoryItem>(request);
        inventoryItem.Id = Guid.NewGuid();
        inventoryItem.QuantityOnHand = request.InitialQuantity;
        inventoryItem.LastReceivedAt = DateTime.UtcNow;

        var created = await _inventoryRepository.CreateAsync(inventoryItem, cancellationToken);

        // Create initial stock movement if quantity > 0
        if (request.InitialQuantity > 0)
        {
            var movement = new StockMovement
            {
                Id = Guid.NewGuid(),
                TenantId = request.TenantId,
                InventoryItemId = created.Id,
                WarehouseId = request.WarehouseId,
                MovementType = MovementType.Receipt,
                Quantity = request.InitialQuantity,
                QuantityBefore = 0,
                QuantityAfter = request.InitialQuantity,
                Reason = "Initial stock",
                MovementDate = DateTime.UtcNow,
                CreatedBy = request.CreatedBy
            };
            await _stockMovementRepository.CreateAsync(movement, cancellationToken);
        }

        _logger.LogInformation("Inventory item created: {InventoryItemId} for Product {ProductId}", created.Id, created.ProductId);

        return await MapToInventoryItemResponseAsync(created, warehouse.Name);
    }

    public async Task<InventoryItemResponse> UpdateInventoryItemAsync(Guid id, UpdateInventoryItemRequest request, CancellationToken cancellationToken = default)
    {
        var inventoryItem = await _inventoryRepository.GetByIdAsync(id, cancellationToken)
            ?? throw new KeyNotFoundException($"Inventory item {id} not found");

        if (request.ReorderPoint.HasValue) inventoryItem.ReorderPoint = request.ReorderPoint.Value;
        if (request.ReorderQuantity.HasValue) inventoryItem.ReorderQuantity = request.ReorderQuantity.Value;
        if (request.MaxStock.HasValue) inventoryItem.MaxStock = request.MaxStock.Value;
        if (request.BinLocation != null) inventoryItem.BinLocation = request.BinLocation;
        inventoryItem.UpdatedBy = request.UpdatedBy;

        var updated = await _inventoryRepository.UpdateAsync(inventoryItem, cancellationToken);
        _logger.LogInformation("Inventory item updated: {InventoryItemId}", id);

        return await MapToInventoryItemResponseAsync(updated, updated.Warehouse.Name);
    }

    public async Task<InventoryItemResponse?> GetInventoryItemByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var inventoryItem = await _inventoryRepository.GetByIdAsync(id, cancellationToken);
        return inventoryItem == null ? null : await MapToInventoryItemResponseAsync(inventoryItem, inventoryItem.Warehouse.Name);
    }

    public async Task<StockLevelResponse?> GetStockLevelAsync(string tenantId, string productId, string? variantId = null, CancellationToken cancellationToken = default)
    {
        var items = await _inventoryRepository.GetByProductIdAsync(tenantId, productId, cancellationToken);

        if (variantId != null)
        {
            items = items.Where(i => i.VariantId == variantId).ToList();
        }

        if (!items.Any())
        {
            return null;
        }

        var response = new StockLevelResponse
        {
            ProductId = productId,
            VariantId = variantId,
            SKU = items.First().SKU,
            TotalOnHand = items.Sum(i => i.QuantityOnHand),
            TotalReserved = items.Sum(i => i.QuantityReserved),
            TotalAvailable = items.Sum(i => i.QuantityAvailable),
            WarehouseLevels = items.Select(i => new WarehouseStockLevel
            {
                WarehouseId = i.WarehouseId,
                WarehouseName = i.Warehouse.Name,
                QuantityOnHand = i.QuantityOnHand,
                QuantityReserved = i.QuantityReserved,
                QuantityAvailable = i.QuantityAvailable
            }).ToList()
        };

        return response;
    }

    public async Task<List<InventoryItemResponse>> GetLowStockItemsAsync(string tenantId, CancellationToken cancellationToken = default)
    {
        var items = await _inventoryRepository.GetLowStockItemsAsync(tenantId, cancellationToken);
        var responses = new List<InventoryItemResponse>();

        foreach (var item in items)
        {
            responses.Add(await MapToInventoryItemResponseAsync(item, item.Warehouse.Name));
        }

        return responses;
    }

    public async Task<(List<InventoryItemResponse> Items, int Total)> GetInventoryItemsPagedAsync(string tenantId, int offset, int limit, CancellationToken cancellationToken = default)
    {
        var (items, total) = await _inventoryRepository.GetPagedAsync(tenantId, offset, limit, cancellationToken);
        var responses = new List<InventoryItemResponse>();

        foreach (var item in items)
        {
            responses.Add(await MapToInventoryItemResponseAsync(item, item.Warehouse.Name));
        }

        return (responses, total);
    }

    public async Task DeleteInventoryItemAsync(Guid id, CancellationToken cancellationToken = default)
    {
        await _inventoryRepository.DeleteAsync(id, cancellationToken);
        _logger.LogInformation("Inventory item deleted: {InventoryItemId}", id);
    }

    #endregion

    #region Stock Operations

    public async Task<InventoryItemResponse> AdjustStockAsync(Guid inventoryItemId, AdjustStockRequest request, CancellationToken cancellationToken = default)
    {
        var inventoryItem = await _inventoryRepository.GetByIdAsync(inventoryItemId, cancellationToken)
            ?? throw new KeyNotFoundException($"Inventory item {inventoryItemId} not found");

        var quantityBefore = inventoryItem.QuantityOnHand;
        var quantityAfter = quantityBefore + request.Quantity;

        if (quantityAfter < 0)
        {
            throw new InvalidOperationException("Adjustment would result in negative stock");
        }

        inventoryItem.QuantityOnHand = quantityAfter;
        inventoryItem.UpdatedBy = request.CreatedBy;
        inventoryItem.LastStockCheckAt = DateTime.UtcNow;

        var movement = new StockMovement
        {
            Id = Guid.NewGuid(),
            TenantId = inventoryItem.TenantId,
            InventoryItemId = inventoryItem.Id,
            WarehouseId = inventoryItem.WarehouseId,
            MovementType = MovementType.Adjustment,
            Quantity = request.Quantity,
            QuantityBefore = quantityBefore,
            QuantityAfter = quantityAfter,
            Reason = request.Reason,
            Notes = request.Notes,
            MovementDate = DateTime.UtcNow,
            CreatedBy = request.CreatedBy
        };

        await _inventoryRepository.UpdateAsync(inventoryItem, cancellationToken);
        await _stockMovementRepository.CreateAsync(movement, cancellationToken);

        _logger.LogInformation("Stock adjusted for inventory item {InventoryItemId}: {Quantity}", inventoryItemId, request.Quantity);

        return await MapToInventoryItemResponseAsync(inventoryItem, inventoryItem.Warehouse.Name);
    }

    public async Task<InventoryItemResponse> TransferStockAsync(Guid inventoryItemId, TransferStockRequest request, CancellationToken cancellationToken = default)
    {
        var sourceItem = await _inventoryRepository.GetByIdAsync(inventoryItemId, cancellationToken)
            ?? throw new KeyNotFoundException($"Source inventory item {inventoryItemId} not found");

        if (sourceItem.QuantityAvailable < request.Quantity)
        {
            throw new InvalidOperationException("Insufficient available stock for transfer");
        }

        var targetWarehouse = await _warehouseRepository.GetByIdAsync(request.ToWarehouseId, cancellationToken)
            ?? throw new KeyNotFoundException($"Target warehouse {request.ToWarehouseId} not found");

        // Get or create target inventory item
        var targetItem = await _inventoryRepository.GetByProductAsync(
            sourceItem.TenantId, request.ToWarehouseId, sourceItem.ProductId, sourceItem.VariantId, cancellationToken);

        if (targetItem == null)
        {
            targetItem = new InventoryItem
            {
                Id = Guid.NewGuid(),
                TenantId = sourceItem.TenantId,
                WarehouseId = request.ToWarehouseId,
                ProductId = sourceItem.ProductId,
                VariantId = sourceItem.VariantId,
                SKU = sourceItem.SKU,
                QuantityOnHand = 0,
                ReorderPoint = sourceItem.ReorderPoint,
                ReorderQuantity = sourceItem.ReorderQuantity,
                CreatedBy = request.CreatedBy
            };
            targetItem = await _inventoryRepository.CreateAsync(targetItem, cancellationToken);
        }

        // Update quantities
        sourceItem.QuantityOnHand -= request.Quantity;
        targetItem.QuantityOnHand += request.Quantity;

        // Create movements
        var outMovement = new StockMovement
        {
            Id = Guid.NewGuid(),
            TenantId = sourceItem.TenantId,
            InventoryItemId = sourceItem.Id,
            WarehouseId = sourceItem.WarehouseId,
            MovementType = MovementType.Transfer,
            Quantity = -request.Quantity,
            QuantityBefore = sourceItem.QuantityOnHand + request.Quantity,
            QuantityAfter = sourceItem.QuantityOnHand,
            FromWarehouseId = sourceItem.WarehouseId,
            ToWarehouseId = request.ToWarehouseId,
            Reason = request.Reason,
            Notes = request.Notes,
            MovementDate = DateTime.UtcNow,
            CreatedBy = request.CreatedBy
        };

        var inMovement = new StockMovement
        {
            Id = Guid.NewGuid(),
            TenantId = targetItem.TenantId,
            InventoryItemId = targetItem.Id,
            WarehouseId = targetItem.WarehouseId,
            MovementType = MovementType.Transfer,
            Quantity = request.Quantity,
            QuantityBefore = targetItem.QuantityOnHand - request.Quantity,
            QuantityAfter = targetItem.QuantityOnHand,
            FromWarehouseId = sourceItem.WarehouseId,
            ToWarehouseId = request.ToWarehouseId,
            Reason = request.Reason,
            Notes = request.Notes,
            MovementDate = DateTime.UtcNow,
            CreatedBy = request.CreatedBy
        };

        await _inventoryRepository.UpdateAsync(sourceItem, cancellationToken);
        await _inventoryRepository.UpdateAsync(targetItem, cancellationToken);
        await _stockMovementRepository.CreateAsync(outMovement, cancellationToken);
        await _stockMovementRepository.CreateAsync(inMovement, cancellationToken);

        _logger.LogInformation("Stock transferred from inventory {SourceId} to warehouse {TargetWarehouseId}: {Quantity}",
            inventoryItemId, request.ToWarehouseId, request.Quantity);

        return await MapToInventoryItemResponseAsync(sourceItem, sourceItem.Warehouse.Name);
    }

    public async Task<StockReservationResponse> ReserveStockAsync(ReserveStockRequest request, CancellationToken cancellationToken = default)
    {
        // Find available inventory across warehouses
        var inventoryItems = await _inventoryRepository.GetByProductIdAsync(request.TenantId, request.ProductId, cancellationToken);

        if (request.VariantId != null)
        {
            inventoryItems = inventoryItems.Where(i => i.VariantId == request.VariantId).ToList();
        }

        var itemWithStock = inventoryItems
            .Where(i => i.QuantityAvailable >= request.Quantity)
            .OrderByDescending(i => i.Warehouse.IsDefault)
            .FirstOrDefault();

        if (itemWithStock == null)
        {
            throw new InvalidOperationException("Insufficient stock available for reservation");
        }

        var reservation = new StockReservation
        {
            Id = Guid.NewGuid(),
            TenantId = request.TenantId,
            InventoryItemId = itemWithStock.Id,
            OrderId = request.OrderId,
            OrderItemId = request.OrderItemId,
            QuantityReserved = request.Quantity,
            Status = ReservationStatus.Active,
            ReservedAt = DateTime.UtcNow,
            ExpiresAt = DateTime.UtcNow.AddMinutes(request.ExpirationMinutes),
            CreatedBy = request.CreatedBy
        };

        itemWithStock.QuantityReserved += request.Quantity;

        await _inventoryRepository.UpdateAsync(itemWithStock, cancellationToken);
        await _stockReservationRepository.CreateAsync(reservation, cancellationToken);

        _logger.LogInformation("Stock reserved: {ReservationId} for Order {OrderId}, Quantity: {Quantity}",
            reservation.Id, request.OrderId, request.Quantity);

        return _mapper.Map<StockReservationResponse>(reservation);
    }

    public async Task<StockReservationResponse> FulfillReservationAsync(Guid reservationId, string fulfilledBy, CancellationToken cancellationToken = default)
    {
        var reservation = await _stockReservationRepository.GetByIdAsync(reservationId, cancellationToken)
            ?? throw new KeyNotFoundException($"Reservation {reservationId} not found");

        if (reservation.Status != ReservationStatus.Active)
        {
            throw new InvalidOperationException($"Reservation is {reservation.Status}, cannot fulfill");
        }

        var inventoryItem = reservation.InventoryItem;

        // Deduct from on-hand and reserved quantities
        inventoryItem.QuantityOnHand -= reservation.QuantityReserved;
        inventoryItem.QuantityReserved -= reservation.QuantityReserved;

        reservation.Status = ReservationStatus.Fulfilled;
        reservation.FulfilledAt = DateTime.UtcNow;
        reservation.UpdatedBy = fulfilledBy;

        // Create shipment movement
        var movement = new StockMovement
        {
            Id = Guid.NewGuid(),
            TenantId = reservation.TenantId,
            InventoryItemId = inventoryItem.Id,
            WarehouseId = inventoryItem.WarehouseId,
            MovementType = MovementType.Shipment,
            Quantity = -reservation.QuantityReserved,
            QuantityBefore = inventoryItem.QuantityOnHand + reservation.QuantityReserved,
            QuantityAfter = inventoryItem.QuantityOnHand,
            OrderId = reservation.OrderId,
            Reason = "Order fulfillment",
            MovementDate = DateTime.UtcNow,
            CreatedBy = fulfilledBy
        };

        await _inventoryRepository.UpdateAsync(inventoryItem, cancellationToken);
        await _stockReservationRepository.UpdateAsync(reservation, cancellationToken);
        await _stockMovementRepository.CreateAsync(movement, cancellationToken);

        _logger.LogInformation("Reservation fulfilled: {ReservationId}", reservationId);

        return _mapper.Map<StockReservationResponse>(reservation);
    }

    public async Task<StockReservationResponse> CancelReservationAsync(Guid reservationId, string cancelledBy, string? reason = null, CancellationToken cancellationToken = default)
    {
        var reservation = await _stockReservationRepository.GetByIdAsync(reservationId, cancellationToken)
            ?? throw new KeyNotFoundException($"Reservation {reservationId} not found");

        if (reservation.Status != ReservationStatus.Active)
        {
            throw new InvalidOperationException($"Reservation is {reservation.Status}, cannot cancel");
        }

        var inventoryItem = reservation.InventoryItem;
        inventoryItem.QuantityReserved -= reservation.QuantityReserved;

        reservation.Status = ReservationStatus.Cancelled;
        reservation.CancelledAt = DateTime.UtcNow;
        reservation.CancellationReason = reason;
        reservation.UpdatedBy = cancelledBy;

        await _inventoryRepository.UpdateAsync(inventoryItem, cancellationToken);
        await _stockReservationRepository.UpdateAsync(reservation, cancellationToken);

        _logger.LogInformation("Reservation cancelled: {ReservationId}", reservationId);

        return _mapper.Map<StockReservationResponse>(reservation);
    }

    public async Task ProcessExpiredReservationsAsync(CancellationToken cancellationToken = default)
    {
        var expiredReservations = await _stockReservationRepository.GetExpiredReservationsAsync(cancellationToken);

        foreach (var reservation in expiredReservations)
        {
            var inventoryItem = reservation.InventoryItem;
            inventoryItem.QuantityReserved -= reservation.QuantityReserved;

            reservation.Status = ReservationStatus.Expired;
            reservation.CancelledAt = DateTime.UtcNow;
            reservation.CancellationReason = "Reservation expired";

            await _inventoryRepository.UpdateAsync(inventoryItem, cancellationToken);
            await _stockReservationRepository.UpdateAsync(reservation, cancellationToken);
        }

        _logger.LogInformation("Processed {Count} expired reservations", expiredReservations.Count);
    }

    #endregion

    #region Stock Movement History

    public async Task<(List<StockMovementResponse> Items, int Total)> GetStockMovementsPagedAsync(string tenantId, int offset, int limit, DateTime? startDate = null, DateTime? endDate = null, CancellationToken cancellationToken = default)
    {
        var (movements, total) = await _stockMovementRepository.GetPagedAsync(tenantId, offset, limit, startDate, endDate, cancellationToken);

        var responses = movements.Select(m => new StockMovementResponse
        {
            Id = m.Id,
            TenantId = m.TenantId,
            InventoryItemId = m.InventoryItemId,
            WarehouseId = m.WarehouseId,
            WarehouseName = m.Warehouse.Name,
            MovementType = m.MovementType.ToString(),
            Quantity = m.Quantity,
            QuantityBefore = m.QuantityBefore,
            QuantityAfter = m.QuantityAfter,
            Reference = m.Reference,
            OrderId = m.OrderId,
            Reason = m.Reason,
            Notes = m.Notes,
            MovementDate = m.MovementDate,
            FromWarehouseId = m.FromWarehouseId,
            ToWarehouseId = m.ToWarehouseId,
            CreatedBy = m.CreatedBy,
            CreatedAt = m.CreatedAt
        }).ToList();

        return (responses, total);
    }

    public async Task<List<StockMovementResponse>> GetStockMovementsByOrderAsync(string orderId, CancellationToken cancellationToken = default)
    {
        var movements = await _stockMovementRepository.GetByOrderAsync(orderId, cancellationToken);

        return movements.Select(m => new StockMovementResponse
        {
            Id = m.Id,
            TenantId = m.TenantId,
            InventoryItemId = m.InventoryItemId,
            WarehouseId = m.WarehouseId,
            WarehouseName = m.Warehouse.Name,
            MovementType = m.MovementType.ToString(),
            Quantity = m.Quantity,
            QuantityBefore = m.QuantityBefore,
            QuantityAfter = m.QuantityAfter,
            Reference = m.Reference,
            OrderId = m.OrderId,
            Reason = m.Reason,
            Notes = m.Notes,
            MovementDate = m.MovementDate,
            FromWarehouseId = m.FromWarehouseId,
            ToWarehouseId = m.ToWarehouseId,
            CreatedBy = m.CreatedBy,
            CreatedAt = m.CreatedAt
        }).ToList();
    }

    #endregion

    #region Helper Methods

    private async Task<InventoryItemResponse> MapToInventoryItemResponseAsync(InventoryItem item, string warehouseName)
    {
        var needsReorder = item.QuantityAvailable <= item.ReorderPoint;

        return new InventoryItemResponse
        {
            Id = item.Id,
            TenantId = item.TenantId,
            WarehouseId = item.WarehouseId,
            WarehouseName = warehouseName,
            ProductId = item.ProductId,
            VariantId = item.VariantId,
            SKU = item.SKU,
            QuantityOnHand = item.QuantityOnHand,
            QuantityReserved = item.QuantityReserved,
            QuantityAvailable = item.QuantityAvailable,
            ReorderPoint = item.ReorderPoint,
            ReorderQuantity = item.ReorderQuantity,
            MaxStock = item.MaxStock,
            BinLocation = item.BinLocation,
            LastStockCheckAt = item.LastStockCheckAt,
            LastReceivedAt = item.LastReceivedAt,
            NeedsReorder = needsReorder,
            CreatedAt = item.CreatedAt,
            UpdatedAt = item.UpdatedAt
        };
    }

    #endregion
}
