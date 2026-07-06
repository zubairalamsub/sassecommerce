using System.Security.Claims;
using Ecommerce.PaymentService.Controllers;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Services;
using FluentAssertions;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Mvc;
using Microsoft.Extensions.Logging;
using Moq;

namespace Ecommerce.PaymentService.Tests;

public class PaymentsControllerTests
{
    private const string TestTenant = "t1";
    private readonly Mock<IPaymentService> _paymentService;
    private readonly PaymentsController _controller;

    public PaymentsControllerTests()
    {
        _paymentService = new Mock<IPaymentService>();
        var logger = new Mock<ILogger<PaymentsController>>();
        _controller = new PaymentsController(_paymentService.Object, logger.Object);
        // Authenticated request carrying the verified tenant claim (as minted by user-service).
        _controller.ControllerContext = BuildContext(TestTenant);
    }

    private static ControllerContext BuildContext(string? tenantId)
    {
        var claims = new List<Claim>();
        if (tenantId != null)
        {
            claims.Add(new Claim("tenant_id", tenantId));
        }
        var identity = new ClaimsIdentity(claims, "TestAuth");
        var principal = new ClaimsPrincipal(identity);
        return new ControllerContext
        {
            HttpContext = new DefaultHttpContext { User = principal }
        };
    }

    private PaymentsController ControllerWithoutTenant()
    {
        var controller = new PaymentsController(_paymentService.Object, new Mock<ILogger<PaymentsController>>().Object);
        controller.ControllerContext = BuildContext(null);
        return controller;
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
            new CreatePaymentRequest { OrderId = "order-1", Amount = 100m, CustomerId = "c1", Method = "credit_card" },
            CancellationToken.None);

        var createdResult = result.Result.Should().BeOfType<CreatedAtActionResult>().Subject;
        createdResult.StatusCode.Should().Be(201);
        var value = createdResult.Value.Should().BeOfType<PaymentResponse>().Subject;
        value.Id.Should().Be(response.Id);
    }

    [Fact]
    public async Task ProcessPayment_ShouldOverrideTenantFromClaim()
    {
        CreatePaymentRequest? captured = null;
        _paymentService.Setup(s => s.ProcessPaymentAsync(It.IsAny<CreatePaymentRequest>(), It.IsAny<CancellationToken>()))
            .Callback<CreatePaymentRequest, CancellationToken>((req, _) => captured = req)
            .ReturnsAsync(new PaymentResponse { Id = Guid.NewGuid(), Status = "Completed" });

        // Client attempts to spoof another tenant in the body; controller must ignore it.
        await _controller.ProcessPayment(
            new CreatePaymentRequest { TenantId = "attacker-tenant", OrderId = "o1", Amount = 10m, CustomerId = "c1", Method = "credit_card" },
            CancellationToken.None);

        captured.Should().NotBeNull();
        captured!.TenantId.Should().Be(TestTenant);
    }

    [Fact]
    public async Task ProcessPayment_MissingTenantClaim_ShouldReturn401()
    {
        var result = await ControllerWithoutTenant().ProcessPayment(
            new CreatePaymentRequest { OrderId = "o1", Amount = 10m, CustomerId = "c1", Method = "credit_card" },
            CancellationToken.None);

        result.Result.Should().BeOfType<UnauthorizedObjectResult>();
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
            new CreatePaymentRequest { OrderId = "order-1", Amount = 100m, CustomerId = "c1", Method = "credit_card" },
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
        _paymentService.Setup(s => s.GetPaymentByIdAsync(id, It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentDetailResponse { Id = id, Status = "Completed" });

        var result = await _controller.GetPayment(id, CancellationToken.None);

        var okResult = result.Result.Should().BeOfType<OkObjectResult>().Subject;
        var value = okResult.Value.Should().BeOfType<PaymentDetailResponse>().Subject;
        value.Id.Should().Be(id);
    }

    [Fact]
    public async Task GetPayment_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetPaymentByIdAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentDetailResponse?)null);

        var result = await _controller.GetPayment(Guid.NewGuid(), CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task GetPayment_MissingTenantClaim_ShouldReturn401()
    {
        var result = await ControllerWithoutTenant().GetPayment(Guid.NewGuid(), CancellationToken.None);

        result.Result.Should().BeOfType<UnauthorizedObjectResult>();
    }

    #endregion

    #region GetPaymentByOrder

    [Fact]
    public async Task GetPaymentByOrder_WhenExists_ShouldReturn200()
    {
        _paymentService.Setup(s => s.GetPaymentByOrderIdAsync(TestTenant, "order-1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentDetailResponse { OrderId = "order-1" });

        var result = await _controller.GetPaymentByOrder("order-1", CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task GetPaymentByOrder_MissingTenantClaim_ShouldReturn401()
    {
        var result = await ControllerWithoutTenant().GetPaymentByOrder("order-1", CancellationToken.None);

        result.Result.Should().BeOfType<UnauthorizedObjectResult>();
    }

    [Fact]
    public async Task GetPaymentByOrder_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetPaymentByOrderIdAsync(TestTenant, "missing", It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentDetailResponse?)null);

        var result = await _controller.GetPaymentByOrder("missing", CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    #endregion

    #region GetPayments (Paged)

    [Fact]
    public async Task GetPayments_MissingTenantClaim_ShouldReturn401()
    {
        var result = await ControllerWithoutTenant().GetPayments(0, 20, null, CancellationToken.None);

        result.Result.Should().BeOfType<UnauthorizedObjectResult>();
    }

    [Fact]
    public async Task GetPayments_ShouldReturnPagedResult()
    {
        var payments = new List<PaymentResponse>
        {
            new() { Id = Guid.NewGuid(), Status = "Completed" }
        };
        _paymentService.Setup(s => s.GetPaymentsPagedAsync(TestTenant, 0, 20, null, It.IsAny<CancellationToken>()))
            .ReturnsAsync((payments, 1));

        var result = await _controller.GetPayments(0, 20, null, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    #endregion

    #region CancelPayment

    [Fact]
    public async Task CancelPayment_Success_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.CancelPaymentAsync(id, It.IsAny<string>(), It.IsAny<CancelPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentResponse { Id = id, Status = "Cancelled" });

        var result = await _controller.CancelPayment(id, new CancelPaymentRequest { Reason = "test" }, CancellationToken.None);

        var okResult = result.Result.Should().BeOfType<OkObjectResult>().Subject;
        var value = okResult.Value.Should().BeOfType<PaymentResponse>().Subject;
        value.Status.Should().Be("Cancelled");
    }

    [Fact]
    public async Task CancelPayment_NotFound_ShouldReturn404()
    {
        _paymentService.Setup(s => s.CancelPaymentAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<CancelPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new KeyNotFoundException());

        var result = await _controller.CancelPayment(Guid.NewGuid(), new CancelPaymentRequest { Reason = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task CancelPayment_InvalidState_ShouldReturn400()
    {
        _paymentService.Setup(s => s.CancelPaymentAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<CancelPaymentRequest>(), It.IsAny<CancellationToken>()))
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
        _paymentService.Setup(s => s.RefundPaymentAsync(id, It.IsAny<string>(), It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RefundResponse { Id = Guid.NewGuid(), Status = "Completed", Amount = 100m });

        var result = await _controller.RefundPayment(id, new RefundPaymentRequest { Reason = "return" }, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task RefundPayment_GatewayFails_ShouldReturn422()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.RefundPaymentAsync(id, It.IsAny<string>(), It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RefundResponse { Status = "Failed", FailureReason = "Gateway error" });

        var result = await _controller.RefundPayment(id, new RefundPaymentRequest { Reason = "return" }, CancellationToken.None);

        result.Result.Should().BeOfType<UnprocessableEntityObjectResult>();
    }

    [Fact]
    public async Task RefundPayment_NotFound_ShouldReturn404()
    {
        _paymentService.Setup(s => s.RefundPaymentAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new KeyNotFoundException());

        var result = await _controller.RefundPayment(Guid.NewGuid(), new RefundPaymentRequest { Reason = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task RefundPayment_InvalidState_ShouldReturn400()
    {
        _paymentService.Setup(s => s.RefundPaymentAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<RefundPaymentRequest>(), It.IsAny<CancellationToken>()))
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
        _paymentService.Setup(s => s.GetRefundByIdAsync(refundId, It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new RefundResponse { Id = refundId });

        var result = await _controller.GetRefund(refundId, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task GetRefund_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetRefundByIdAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<CancellationToken>()))
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
        _paymentService.Setup(s => s.GetRefundsByPaymentAsync(paymentId, It.IsAny<string>(), It.IsAny<CancellationToken>()))
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
            new CreatePaymentMethodRequest { CustomerId = "c1", Type = "credit_card" },
            CancellationToken.None);

        result.Result.Should().BeOfType<CreatedAtActionResult>();
    }

    [Fact]
    public async Task CreatePaymentMethod_ShouldOverrideTenantFromClaim()
    {
        CreatePaymentMethodRequest? captured = null;
        _paymentService.Setup(s => s.CreatePaymentMethodAsync(It.IsAny<CreatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .Callback<CreatePaymentMethodRequest, CancellationToken>((req, _) => captured = req)
            .ReturnsAsync(new PaymentMethodResponse { Id = Guid.NewGuid(), Type = "CreditCard" });

        await _controller.CreatePaymentMethod(
            new CreatePaymentMethodRequest { TenantId = "attacker-tenant", CustomerId = "c1", Type = "credit_card" },
            CancellationToken.None);

        captured.Should().NotBeNull();
        captured!.TenantId.Should().Be(TestTenant);
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
        _paymentService.Setup(s => s.GetPaymentMethodByIdAsync(id, It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentMethodResponse { Id = id });

        var result = await _controller.GetPaymentMethod(id, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task GetPaymentMethod_WhenNotExists_ShouldReturn404()
    {
        _paymentService.Setup(s => s.GetPaymentMethodByIdAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethodResponse?)null);

        var result = await _controller.GetPaymentMethod(Guid.NewGuid(), CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task GetPaymentMethods_MissingCustomerId_ShouldReturn400()
    {
        var result = await _controller.GetPaymentMethods("", CancellationToken.None);

        result.Result.Should().BeOfType<BadRequestObjectResult>();
    }

    [Fact]
    public async Task GetPaymentMethods_MissingTenantClaim_ShouldReturn401()
    {
        var result = await ControllerWithoutTenant().GetPaymentMethods("c1", CancellationToken.None);

        result.Result.Should().BeOfType<UnauthorizedObjectResult>();
    }

    [Fact]
    public async Task GetPaymentMethods_Valid_ShouldReturn200()
    {
        _paymentService.Setup(s => s.GetPaymentMethodsByCustomerAsync(TestTenant, "c1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new List<PaymentMethodResponse>());

        var result = await _controller.GetPaymentMethods("c1", CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task UpdatePaymentMethod_Success_ShouldReturn200()
    {
        var id = Guid.NewGuid();
        _paymentService.Setup(s => s.UpdatePaymentMethodAsync(id, It.IsAny<string>(), It.IsAny<UpdatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new PaymentMethodResponse { Id = id });

        var result = await _controller.UpdatePaymentMethod(id, new UpdatePaymentMethodRequest { UpdatedBy = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    [Fact]
    public async Task UpdatePaymentMethod_NotFound_ShouldReturn404()
    {
        _paymentService.Setup(s => s.UpdatePaymentMethodAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<UpdatePaymentMethodRequest>(), It.IsAny<CancellationToken>()))
            .ThrowsAsync(new KeyNotFoundException());

        var result = await _controller.UpdatePaymentMethod(Guid.NewGuid(), new UpdatePaymentMethodRequest { UpdatedBy = "test" }, CancellationToken.None);

        result.Result.Should().BeOfType<NotFoundResult>();
    }

    [Fact]
    public async Task DeletePaymentMethod_ShouldReturn204()
    {
        _paymentService.Setup(s => s.DeletePaymentMethodAsync(It.IsAny<Guid>(), It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .Returns(Task.CompletedTask);

        var result = await _controller.DeletePaymentMethod(Guid.NewGuid(), CancellationToken.None);

        result.Should().BeOfType<NoContentResult>();
    }

    #endregion

    #region GetPaymentsByCustomer

    [Fact]
    public async Task GetPaymentsByCustomer_MissingTenantClaim_ShouldReturn401()
    {
        var result = await ControllerWithoutTenant().GetPaymentsByCustomer("cust-1", CancellationToken.None);

        result.Result.Should().BeOfType<UnauthorizedObjectResult>();
    }

    [Fact]
    public async Task GetPaymentsByCustomer_Valid_ShouldReturn200()
    {
        _paymentService.Setup(s => s.GetPaymentsByCustomerAsync(TestTenant, "c1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new List<PaymentResponse>());

        var result = await _controller.GetPaymentsByCustomer("c1", CancellationToken.None);

        result.Result.Should().BeOfType<OkObjectResult>();
    }

    #endregion
}
