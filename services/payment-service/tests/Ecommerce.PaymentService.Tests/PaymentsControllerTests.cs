using Ecommerce.PaymentService.Controllers;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Services;
using FluentAssertions;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Logging;
using Moq;

namespace Ecommerce.PaymentService.Tests;

public class PaymentsControllerTests
{
    private readonly Mock<IPaymentService> _paymentService;
    private readonly PaymentsController _controller;

    public PaymentsControllerTests()
    {
        _paymentService = new Mock<IPaymentService>();
        var logger = new Mock<ILogger<PaymentsController>>();
        _controller = new PaymentsController(_paymentService.Object, logger.Object);
    }

    #region ProcessPayment

    [Fact]
    public async Task ProcessPayment_Success_ShouldReturn201Created()
    {
        var response = new PaymentResponse
        {
            Id = Guid.NewGuid(),
            Status = "Completed",
            Amount = 100m,
            OrderId = "order-1"
        };

        _paymentService.Setup(s => s.ProcessPaymentAsync(It.IsAny<CreatePaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(response);

        var result = await _controller.ProcessPayment(
            new CreatePaymentRequest { TenantId = "t1", OrderId = "order-1", Amount = 100m, CustomerId = "c1", Method = "credit_card" },
            CancellationToken.None);

        var createdResult = result.Result.Should().BeOfType<CreatedAtActionResult>().Subject;
        createdResult.StatusCode.Should().Be(201);
        var value = createdResult.Value.Should().BeOfType<PaymentResponse>().Subject;
        value.Id.Should().Be(response.Id);
    }

    [Fact]
    public async Task ProcessPayment_GatewayFails_ShouldReturn422()
    {
        var response = new PaymentResponse
        {
            Id = Guid.NewGuid(),
            Status = "Failed",
            FailureReason = "Card declined"
        };

        _paymentService.Setup(s => s.ProcessPaymentAsync(It.IsAny<CreatePaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(response);

        var result = await _controller.ProcessPayment(
            new CreatePaymentRequest { TenantId = "t1", OrderId = "order-1", Amount = 100m, CustomerId = "c1", Method = "credit_card" },
            CancellationToken.None);

        var objectResult = result.Result.Should().BeOfType<UnprocessableEntityObjectResult>().Subject;
        objectResult.StatusCode.Should().Be(422);
    }

    [Fact]
    public async Task ProcessPayment_InvalidOperation_ShouldReturn400()
    {
        _paymentService.Setup(s => s.ProcessPaymentAsync(It.IsAny<CreatePaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new InvalidOperationException("Bad request"));

        var result = await _controller.ProcessPayment(
            new CreatePaymentRequest(),
            CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    #endregion

    #region GetPayment

    [Fact]
    public async Task GetPayment_WhenExists_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.GetPaymentByIdAsync(id, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentDetailResponse { Id = id, Status = "Completed" });

        var result = await _controller.GetPayment(id, CancellationToken.None);

        var okResult = result.Result.Should().BeOfType<OkObjectResult>().Subject;
        var value = okResult.Value.Should().BeOfType<PaymentDetailResponse>().Subject;
        value.Id.Should().Be(id);
    }

    [Fact]
    public async Task GetPayment_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetPaymentByIdAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentDetailResponse?)null);

        var result = await _controller.GetPayment(Guid.NewGuid(), CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    #endregion

    #region GetPaymentByOrder

    [Fact]
    public async Task GetPaymentByOrder_WhenExists_ShouldReturn200()
    {
        _paymentService.Setup(s => s.GetPaymentByOrderIdAsync("t1", "order-1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentDetailResponse { OrderId = "order-1" });

        var result = await _controller.GetPaymentByOrder("order-1", "t1", CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task GetPaymentByOrder_MissingTenantId_ShouldReturn400()
    {
        var result = await _controller.GetPaymentByOrder("order-1", "", CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    [Fact]
    public async Task GetPaymentByOrder_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetPaymentByOrderIdAsync("t1", "missing", It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentDetailResponse?)null);

        var result = await _controller.GetPaymentByOrder("missing", "t1", CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    #endregion

    #region GetPayments (Paged)

    [Fact]
    public async Task GetPayments_MissingTenantId_ShouldReturn400()
    {
        var result = await _controller.GetPayments("", 0, 20, null, CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    [Fact]
    public async Task GetPayments_ShouldReturnPagedResult()
    {
        var payments = new List<PaymentResponse>
        {
            new() { Id = Guid.NewGuid(), Status = "Completed" }
        };
        _paymentService.Setup(s => s.GetPaymentsPagedAsync("t1", 0, 20, null, It.IsAny<CancellationToken>()))
            .ReturnsAsync((payments, 1));

        var result = await _controller.GetPayments("t1", 0, 20, null, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    #endregion

    #region CancelPayment

    [Fact]
    public async Task CancelPayment_Success_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.CancelPaymentAsync(id, It.IsAny<CancelPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentResponse { Id = id, Status = "Cancelled" });

        var result = await _controller.CancelPayment(id, new CancelPaymentRequest { Reason = "test" }, CancellationToken.None);

        var okResult = result.Result.Should().BeOfType<OkObjectResult>().Subject;
        var value = okResult.Value.Should().BeOfType<PaymentResponse>().Subject;
        value.Status.Should().Be("Cancelled");
    }

    [Fact]
    public async Task CancelPayment_NotFound_ShouldReturn404()
    {
        _paymentService.Setup(s => s.CancelPaymentAsync(It.IsAny<Guid>(), It.IsAny<CancelPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new KeyNotFoundException());

        var result = await _controller.CancelPayment(Guid.NewGuid(), new CancelPaymentRequest { Reason = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task CancelPayment_InvalidState_ShouldReturn400()
    {
        _paymentService.Setup(s => s.CancelPaymentAsync(It.IsAny<Guid>(), It.IsAny<CancelPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new InvalidOperationException("Cannot cancel"));

        var result = await _controller.CancelPayment(Guid.NewGuid(), new CancelPaymentRequest { Reason = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    #endregion

    #region RefundPayment

    [Fact]
    public async Task RefundPayment_Success_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.RefundPaymentAsync(id, It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RefundResponse { Id = Guid.NewGuid(), Status = "Completed", Amount = 100m });

        var result = await _controller.RefundPayment(id, new RefundPaymentRequest { Reason = "return" }, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task RefundPayment_GatewayFails_ShouldReturn422()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.RefundPaymentAsync(id, It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RefundResponse { Status = "Failed", FailureReason = "Gateway error" });

        var result = await _controller.RefundPayment(id, new RefundPaymentRequest { Reason = "return" }, CancellationToken.None);

        result.Result.Should().BeOfType<UnprocessableEntityObjectResult>();
    }

    [Fact]
    public async Task RefundPayment_NotFound_ShouldReturn404()
    {
        _paymentService.Setup(s => s.RefundPaymentAsync(It.IsAny<Guid>(), It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new KeyNotFoundException());

        var result = await _controller.RefundPayment(Guid.NewGuid(), new RefundPaymentRequest { Reason = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task RefundPayment_InvalidState_ShouldReturn400()
    {
        _paymentService.Setup(s => s.RefundPaymentAsync(It.IsAny<Guid>(), It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new InvalidOperationException("Cannot refund"));

        var result = await _controller.RefundPayment(Guid.NewGuid(), new RefundPaymentRequest { Reason = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    #endregion

    #region GetRefund

    [Fact]
    public async Task GetRefund_WhenExists_ShouldReturn200()
    {
        var refundId = Guid.NewGuid();
        _paymentService.Setup(s => s.GetRefundByIdAsync(refundId, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RefundResponse { Id = refundId });

        var result = await _controller.GetRefund(refundId, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task GetRefund_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetRefundByIdAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((RefundResponse?)null);

        var result = await _controller.GetRefund(Guid.NewGuid(), CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    #endregion

    #region GetRefundsByPayment

    [Fact]
    public async Task GetRefundsByPayment_ShouldReturn200()
    {
        var paymentId = Guid.NewGuid();
        _paymentService.Setup(s => s.GetRefundsByPaymentAsync(paymentId, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new List<RefundResponse> { new() { Id = Guid.NewGuid() } });

        var result = await _controller.GetRefundsByPayment(paymentId, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    #endregion

    #region Payment Method Endpoints

    [Fact]
    public async Task CreatePaymentMethod_Success_ShouldReturn201()
    {
        var response = new PaymentMethodResponse { Id = Guid.NewGuid(), Type = "CreditCard" };
        _paymentService.Setup(s => s.CreatePaymentMethodAsync(It.IsAny<CreatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(response);

        var result = await _controller.CreatePaymentMethod(
            new CreatePaymentMethodRequest { TenantId = "t1", CustomerId = "c1", Type = "credit_card" },
            CancellationToken.None);

        result.Result.Should().BeOfType<CreatedAtActionResult>();
    }

    [Fact]
    public async Task CreatePaymentMethod_InvalidOperation_ShouldReturn400()
    {
        _paymentService.Setup(s => s.CreatePaymentMethodAsync(It.IsAny<CreatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new InvalidOperationException("Invalid"));

        var result = await _controller.CreatePaymentMethod(new CreatePaymentMethodRequest(), CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    [Fact]
    public async Task GetPaymentMethod_WhenExists_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.GetPaymentMethodByIdAsync(id, It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentMethodResponse { Id = id });

        var result = await _controller.GetPaymentMethod(id, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task GetPaymentMethod_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetPaymentMethodByIdAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethodResponse?)null);

        var result = await _controller.GetPaymentMethod(Guid.NewGuid(), CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task GetPaymentMethods_MissingParams_ShouldReturn400()
    {
        var result = await _controller.GetPaymentMethods("", "", CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    [Fact]
    public async Task GetPaymentMethods_Valid_ShouldReturn200()
    {
        _paymentService.Setup(s => s.GetPaymentMethodsByCustomerAsync("t1", "c1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new List<PaymentMethodResponse>());

        var result = await _controller.GetPaymentMethods("t1", "c1", CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task UpdatePaymentMethod_Success_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.UpdatePaymentMethodAsync(id, It.IsAny<UpdatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentMethodResponse { Id = id });

        var result = await _controller.UpdatePaymentMethod(id, new UpdatePaymentMethodRequest { UpdatedBy = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task UpdatePaymentMethod_NotFound_ShouldReturn404()
    {
        _paymentService.Setup(s => s.UpdatePaymentMethodAsync(It.IsAny<Guid>(), It.IsAny<UpdatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new KeyNotFoundException());

        var result = await _controller.UpdatePaymentMethod(Guid.NewGuid(), new UpdatePaymentMethodRequest { UpdatedBy = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task DeletePaymentMethod_ShouldReturn204()
    {
        _paymentService.Setup(s => s.DeletePaymentMethodAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .Returns(Task.CompletedTask);

        var result = await _controller.DeletePaymentMethod(Guid.NewGuid(), CancellationToken.None);

        result.Should().BeOfType<NoContentResult>();
    }

    #endregion

    #region GetPaymentsByCustomer

    [Fact]
    public async Task GetPaymentsByCustomer_MissingTenantId_ShouldReturn400()
    {
        var result = await _controller.GetPaymentsByCustomer("cust-1", "", CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    [Fact]
    public async Task GetPaymentsByCustomer_Valid_ShouldReturn200()
    {
        _paymentService.Setup(s => s.GetPaymentsByCustomerAsync("t1", "c1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new List<PaymentResponse>());

        var result = await _controller.GetPaymentsByCustomer("c1", "t1", CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    #endregion
}
