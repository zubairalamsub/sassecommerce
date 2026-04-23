using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Services;
using Microsoft.AspNetCore.Mvc;

namespace Ecommerce.PaymentService.Controllers;

[ApiController]
[Route("api/v1/[controller]")]
public class PaymentsController : ControllerBase
{
    private readonly IPaymentService _paymentService;
    private readonly ILogger<PaymentsController> _logger;

    public PaymentsController(IPaymentService paymentService, ILogger<PaymentsController> logger)
    {
        _paymentService = paymentService;
        _logger = logger;
    }

    #region Payment Endpoints

    /// <summary>
    /// Process a payment. Called by Order Service saga.
    /// </summary>
    [HttpPost]
    public async Task<ActionResult<PaymentResponse>> ProcessPayment([FromBody] CreatePaymentRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _paymentService.ProcessPaymentAsync(request, cancellationToken);

            if (result.Status == "Failed")
            {
                return UnprocessableEntity(new { error = result.FailureReason, payment = result });
            }

            return CreatedAtAction(nameof(GetPayment), new { id = result.Id }, result);
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    /// <summary>
    /// Get payment by ID with transaction and refund details.
    /// </summary>
    [HttpGet("{id:guid}")]
    public async Task<ActionResult<PaymentDetailResponse>> GetPayment(Guid id, CancellationToken cancellationToken)
    {
        var result = await _paymentService.GetPaymentByIdAsync(id, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    /// <summary>
    /// Get payment by order ID.
    /// </summary>
    [HttpGet("order/{orderId}")]
    public async Task<ActionResult<PaymentDetailResponse>> GetPaymentByOrder(
        string orderId,
        [FromQuery] string tenantId,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var result = await _paymentService.GetPaymentByOrderIdAsync(tenantId, orderId, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    /// <summary>
    /// List payments for a tenant with optional status filter.
    /// </summary>
    [HttpGet]
    public async Task<ActionResult<object>> GetPayments(
        [FromQuery] string tenantId,
        [FromQuery] int offset = 0,
        [FromQuery] int limit = 20,
        [FromQuery] string? status = null,
        CancellationToken cancellationToken = default)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var (items, total) = await _paymentService.GetPaymentsPagedAsync(tenantId, offset, limit, status, cancellationToken);
        return Ok(new { data = items, total, offset, limit });
    }

    /// <summary>
    /// Get payments for a specific customer.
    /// </summary>
    [HttpGet("customer/{customerId}")]
    public async Task<ActionResult<List<PaymentResponse>>> GetPaymentsByCustomer(
        string customerId,
        [FromQuery] string tenantId,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrEmpty(tenantId))
        {
            return BadRequest(new { error = "TenantId is required" });
        }

        var result = await _paymentService.GetPaymentsByCustomerAsync(tenantId, customerId, cancellationToken);
        return Ok(result);
    }

    /// <summary>
    /// Cancel a pending/processing payment.
    /// </summary>
    [HttpPost("{id:guid}/cancel")]
    public async Task<ActionResult<PaymentResponse>> CancelPayment(Guid id, [FromBody] CancelPaymentRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _paymentService.CancelPaymentAsync(id, request, cancellationToken);
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

    #region Refund Endpoints

    /// <summary>
    /// Refund a payment (full or partial). Called by Order Service saga for compensation.
    /// </summary>
    [HttpPost("{id:guid}/refund")]
    public async Task<ActionResult<RefundResponse>> RefundPayment(Guid id, [FromBody] RefundPaymentRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _paymentService.RefundPaymentAsync(id, request, cancellationToken);

            if (result.Status == "Failed")
            {
                return UnprocessableEntity(new { error = result.FailureReason, refund = result });
            }

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

    /// <summary>
    /// Get refund details by ID.
    /// </summary>
    [HttpGet("refunds/{refundId:guid}")]
    public async Task<ActionResult<RefundResponse>> GetRefund(Guid refundId, CancellationToken cancellationToken)
    {
        var result = await _paymentService.GetRefundByIdAsync(refundId, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    /// <summary>
    /// List refunds for a specific payment.
    /// </summary>
    [HttpGet("{id:guid}/refunds")]
    public async Task<ActionResult<List<RefundResponse>>> GetRefundsByPayment(Guid id, CancellationToken cancellationToken)
    {
        var result = await _paymentService.GetRefundsByPaymentAsync(id, cancellationToken);
        return Ok(result);
    }

    #endregion

    #region Payment Method Endpoints

    /// <summary>
    /// Add a new payment method for a customer.
    /// </summary>
    [HttpPost("methods")]
    public async Task<ActionResult<PaymentMethodResponse>> CreatePaymentMethod([FromBody] CreatePaymentMethodRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _paymentService.CreatePaymentMethodAsync(request, cancellationToken);
            return CreatedAtAction(nameof(GetPaymentMethod), new { methodId = result.Id }, result);
        }
        catch (InvalidOperationException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    /// <summary>
    /// Get payment method by ID.
    /// </summary>
    [HttpGet("methods/{methodId:guid}")]
    public async Task<ActionResult<PaymentMethodResponse>> GetPaymentMethod(Guid methodId, CancellationToken cancellationToken)
    {
        var result = await _paymentService.GetPaymentMethodByIdAsync(methodId, cancellationToken);
        return result == null ? NotFound() : Ok(result);
    }

    /// <summary>
    /// List payment methods for a customer.
    /// </summary>
    [HttpGet("methods")]
    public async Task<ActionResult<List<PaymentMethodResponse>>> GetPaymentMethods(
        [FromQuery] string tenantId,
        [FromQuery] string customerId,
        CancellationToken cancellationToken)
    {
        if (string.IsNullOrEmpty(tenantId) || string.IsNullOrEmpty(customerId))
        {
            return BadRequest(new { error = "TenantId and CustomerId are required" });
        }

        var result = await _paymentService.GetPaymentMethodsByCustomerAsync(tenantId, customerId, cancellationToken);
        return Ok(result);
    }

    /// <summary>
    /// Update a payment method.
    /// </summary>
    [HttpPut("methods/{methodId:guid}")]
    public async Task<ActionResult<PaymentMethodResponse>> UpdatePaymentMethod(Guid methodId, [FromBody] UpdatePaymentMethodRequest request, CancellationToken cancellationToken)
    {
        try
        {
            var result = await _paymentService.UpdatePaymentMethodAsync(methodId, request, cancellationToken);
            return Ok(result);
        }
        catch (KeyNotFoundException)
        {
            return NotFound();
        }
    }

    /// <summary>
    /// Delete (deactivate) a payment method.
    /// </summary>
    [HttpDelete("methods/{methodId:guid}")]
    public async Task<IActionResult> DeletePaymentMethod(Guid methodId, CancellationToken cancellationToken)
    {
        await _paymentService.DeletePaymentMethodAsync(methodId, cancellationToken);
        return NoContent();
    }

    #endregion
}
