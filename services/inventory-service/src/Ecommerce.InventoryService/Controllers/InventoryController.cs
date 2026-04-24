using Ecommerce.InventoryService.DTOs;
using Ecommerce.InventoryService.Services;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace Ecommerce.InventoryService.Controllers;

[Authorize]
[ApiController]
[Route("api/v1/[controller]")]
public class InventoryController : ControllerBase
{
    private readonly IInventoryService _inventoryService;
    private readonly ILogger<InventoryController> _logger;

    public InventoryController(IInventoryService inventoryService, ILogger<InventoryController> logger)
    {
        _inventoryService = inventoryService;
        _logger = logger;
    }

    #region Warehouse Endpoints

    [HttpPost("warehouses")]
    public async Task<ActionResult<WarehouseResponse>> CreateWarehouse([FromBody] CreateWarehouseRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.CreateWarehouseAsync(request, cancellationToken);
            return CreatedAtAction(nameof(GetWarehouse), new { id = result.Id }, result);
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpGet("warehouses/{id:guid}")]
    public async Task<ActionResult<WarehouseResponse>> GetWarehouse(Guid id, CancellationToken cancellationToken)
    {
        var result = await _inventoryService.GetWarehouseByIdAsync(id, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    [HttpGet("warehouses")]
    public async Task<ActionResult<object>> GetWarehouses([FromQuery] string tenantId, [FromQuery] int offset = 0, [FromQuery] int limit = 20, CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var (items, total) = await _inventoryService.GetWarehousesPagedAsync(tenantId, offset, limit, cancellationToken);
        return Ok(new { data = items, total, offset, limit });
    }

    [HttpPut("warehouses/{id:guid}")]
    public async Task<ActionResult<WarehouseResponse>> UpdateWarehouse(Guid id, [FromBody] UpdateWarehouseRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.UpdateWarehouseAsync(id, request, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }
    }

    [HttpDelete("warehouses/{id:guid}")]
    public async Task<IActionResult> DeleteWarehouse(Guid id, CancellationToken cancellationToken)
    {
        await _inventoryService.DeleteWarehouseAsync(id, cancellationToken);
        return NoContent();
    }

    #endregion

    #region Inventory Item Endpoints

    [HttpPost("items")]
    public async Task<ActionResult<InventoryItemResponse>> CreateInventoryItem([FromBody] CreateInventoryItemRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.CreateInventoryItemAsync(request, cancellationToken);
            return CreatedAtAction(nameof(GetInventoryItem), new { id = result.Id }, result);
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
        catch (KeyNotFoundException ex)
        {
            return NotFound(new { error = ex.Message });
        }
    }

    [HttpGet("items/{id:guid}")]
    public async Task<ActionResult<InventoryItemResponse>> GetInventoryItem(Guid id, CancellationToken cancellationToken)
    {
        var result = await _inventoryService.GetInventoryItemByIdAsync(id, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    [HttpGet("items")]
    public async Task<ActionResult<object>> GetInventoryItems([FromQuery] string tenantId, [FromQuery] int offset = 0, [FromQuery] int limit = 20, CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var (items, total) = await _inventoryService.GetInventoryItemsPagedAsync(tenantId, offset, limit, cancellationToken);
        return Ok(new { data = items, total, offset, limit });
    }

    [HttpGet("items/low-stock")]
    public async Task<ActionResult<List<InventoryItemResponse>>> GetLowStockItems([FromQuery] string tenantId, CancellationToken cancellationToken)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var items = await _inventoryService.GetLowStockItemsAsync(tenantId, cancellationToken);
        return Ok(items);
    }

    [HttpPut("items/{id:guid}")]
    public async Task<ActionResult<InventoryItemResponse>> UpdateInventoryItem(Guid id, [FromBody] UpdateInventoryItemRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.UpdateInventoryItemAsync(id, request, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }
    }

    [HttpDelete("items/{id:guid}")]
    public async Task<IActionResult> DeleteInventoryItem(Guid id, CancellationToken cancellationToken)
    {
        await _inventoryService.DeleteInventoryItemAsync(id, cancellationToken);
        return NoContent();
    }

    #endregion

    #region Stock Level Endpoints

    [HttpGet("stock-levels")]
    public async Task<ActionResult<StockLevelResponse>> GetStockLevel([FromQuery] string tenantId, [FromQuery] string productId, [FromQuery] string? variantId = null, CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrEmpty(tenantId) || string.IsNullOrEmpty(productId))
        {
            return BadRequest(new { error = "TenantId and ProductId are required" });
        }

        var result = await _inventoryService.GetStockLevelAsync(tenantId, productId, variantId, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    #endregion

    #region Stock Operation Endpoints

    [HttpPost("items/{id:guid}/adjust")]
    public async Task<ActionResult<InventoryItemResponse>> AdjustStock(Guid id, [FromBody] AdjustStockRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.AdjustStockAsync(id, request, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpPost("items/{id:guid}/transfer")]
    public async Task<ActionResult<InventoryItemResponse>> TransferStock(Guid id, [FromBody] TransferStockRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.TransferStockAsync(id, request, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException ex)
        {
            return NotFound(new { error = ex.Message });
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpPost("reservations")]
    public async Task<ActionResult<StockReservationResponse>> ReserveStock([FromBody] ReserveStockRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.ReserveStockAsync(request, cancellationToken);
            return CreatedAtAction(nameof(GetReservation), new { id = result.Id }, result);
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpGet("reservations/{id:guid}")]
    public async Task<ActionResult<StockReservationResponse>> GetReservation(Guid id)
    {
        // This is a placeholder - you'd implement GetReservationByIdAsync in the service
        return NotFound();
    }

    [HttpPost("reservations/{id:guid}/fulfill")]
    public async Task<ActionResult<StockReservationResponse>> FulfillReservation(Guid id, [FromBody] FulfillReservationRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.FulfillReservationAsync(id, request.FulfilledBy, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpPost("reservations/{id:guid}/cancel")]
    public async Task<ActionResult<StockReservationResponse>> CancelReservation(Guid id, [FromBody] CancelReservationRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _inventoryService.CancelReservationAsync(id, request.CancelledBy, request.Reason, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    #endregion

    #region Stock Movement Endpoints

    [HttpGet("movements")]
    public async Task<ActionResult<object>> GetStockMovements(
        [FromQuery] string tenantId,
        [FromQuery] int offset = 0,
        [FromQuery] int limit = 20,
        [FromQuery] DateTime? startDate = null,
        [FromQuery] DateTime? endDate = null,
        CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var (items, total) = await _inventoryService.GetStockMovementsPagedAsync(tenantId, offset, limit, startDate, endDate, cancellationToken);
        return Ok(new { data = items, total, offset, limit });
    }

    [HttpGet("movements/order/{orderId}")]
    public async Task<ActionResult<List<StockMovementResponse>>> GetStockMovementsByOrder(string orderId, CancellationToken cancellationToken)
    {
        var items = await _inventoryService.GetStockMovementsByOrderAsync(orderId, cancellationToken);
        return Ok(items);
    }

    #endregion
}

public class FulfillReservationRequest
{
    public string FulfilledBy { get; set; } = string.Empty;
}

public class CancelReservationRequest
{
    public string CancelledBy { get; set; } = string.Empty;
    public string? Reason { get; set; }
}
