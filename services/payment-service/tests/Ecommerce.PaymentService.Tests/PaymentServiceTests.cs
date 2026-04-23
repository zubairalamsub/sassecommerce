using AutoMapper;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Entities;
using Ecommerce.PaymentService.Repositories;
using Ecommerce.PaymentService.Services;
using FluentAssertions;
using Microsoft.Extensions.Logging;
using Moq;

namespace Ecommerce.PaymentService.Tests;

public class PaymentServiceTests
{
    private readonly Mock<IPaymentRepository> _paymentRepo;
    private readonly Mock<IPaymentMethodRepository> _methodRepo;
    private readonly Mock<IPaymentTransactionRepository> _transactionRepo;
    private readonly Mock<IRefundRepository> _refundRepo;
    private readonly Mock<IPaymentGateway> _gateway;
    private readonly IMapper _mapper;
    private readonly Services.PaymentService _service;

    public PaymentServiceTests()
    {
        _paymentRepo = new Mock<IPaymentRepository>();
        _methodRepo = new Mock<IPaymentMethodRepository>();
        _transactionRepo = new Mock<IPaymentTransactionRepository>();
        _refundRepo = new Mock<IRefundRepository>();
        _gateway = new Mock<IPaymentGateway>();
        _mapper = TestHelpers.CreateMapper();

        _gateway.Setup(g => g.Name).Returns("TestGateway");

        _service = new Services.PaymentService(
            _paymentRepo.Object,
            _methodRepo.Object,
            _transactionRepo.Object,
            _refundRepo.Object,
            _gateway.Object,
            _mapper,
            new Mock<ILogger<Services.PaymentService>>().Object
        );
    }

    #region ProcessPayment Tests

    [Fact]
    public async Task ProcessPayment_WithValidRequest_ShouldReturnCompletedPayment()
    {
        // Arrange
        var request = new CreatePaymentRequest
        {
            TenantId = "tenant-1",
            CustomerId = "cust-1",
            OrderId = "order-1",
            Amount = 100.00m,
            Currency = "USD",
            Method = "credit_card",
            CreatedBy = "test"
        };

        _gateway.Setup(g => g.ChargeAsync(It.IsAny<GatewayChargeRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse
            {
                Success = true,
                TransactionId = "gw_txn_123",
                RawResponse = "{\"status\":\"ok\"}"
            });

        _paymentRepo.Setup(r => r.CreateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);

        // Act
        var result = await _service.ProcessPaymentAsync(request);

        // Assert
        result.Should().NotBeNull();
        result.Status.Should().Be("Completed");
        result.Amount.Should().Be(100.00m);
        result.Currency.Should().Be("USD");
        result.OrderId.Should().Be("order-1");
        result.CustomerId.Should().Be("cust-1");
        result.GatewayTransactionId.Should().Be("gw_txn_123");

        _paymentRepo.Verify(r => r.CreateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()), Times.Once);
        _paymentRepo.Verify(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()), Times.Once);
        _transactionRepo.Verify(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()), Times.Once);
    }

    [Fact]
    public async Task ProcessPayment_WhenGatewayFails_ShouldReturnFailedPayment()
    {
        var request = new CreatePaymentRequest
        {
            TenantId = "tenant-1",
            CustomerId = "cust-1",
            OrderId = "order-2",
            Amount = 50.00m,
            Currency = "USD",
            Method = "credit_card",
            CreatedBy = "test"
        };

        _gateway.Setup(g => g.ChargeAsync(It.IsAny<GatewayChargeRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse
            {
                Success = false,
                TransactionId = "",
                ErrorCode = "card_declined",
                ErrorMessage = "Your card was declined"
            });

        _paymentRepo.Setup(r => r.CreateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);

        var result = await _service.ProcessPaymentAsync(request);

        result.Status.Should().Be("Failed");
        result.FailureReason.Should().Be("Your card was declined");
    }

    [Fact]
    public async Task ProcessPayment_WithIdempotencyKey_ShouldReturnExistingPayment()
    {
        var existingPayment = new Payment
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant-1",
            OrderId = "order-1",
            CustomerId = "cust-1",
            Amount = 100m,
            Currency = "USD",
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard,
            IdempotencyKey = "idem-key-1"
        };

        _paymentRepo.Setup(r => r.GetByIdempotencyKeyAsync("idem-key-1", It.IsAny<CancellationToken>()))
            .ReturnsAsync(existingPayment);

        var request = new CreatePaymentRequest
        {
            TenantId = "tenant-1",
            CustomerId = "cust-1",
            OrderId = "order-1",
            Amount = 100m,
            Currency = "USD",
            Method = "credit_card",
            IdempotencyKey = "idem-key-1"
        };

        var result = await _service.ProcessPaymentAsync(request);

        result.Id.Should().Be(existingPayment.Id);
        result.Status.Should().Be("Completed");
        _gateway.Verify(g => g.ChargeAsync(It.IsAny<GatewayChargeRequest>(), It.IsAny<CancellationToken>()), Times.Never);
    }

    [Fact]
    public async Task ProcessPayment_WithPaymentMethodId_ShouldUseStoredToken()
    {
        var methodId = Guid.NewGuid();
        var storedMethod = new PaymentMethod
        {
            Id = methodId,
            Token = "tok_stored_123",
            Type = PaymentMethodType.CreditCard
        };

        _methodRepo.Setup(r => r.GetByIdAsync(methodId, It.IsAny<CancellationToken>()))
            .ReturnsAsync(storedMethod);

        _gateway.Setup(g => g.ChargeAsync(It.Is<GatewayChargeRequest>(r => r.Token == "tok_stored_123"), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse { Success = true, TransactionId = "gw_123" });

        _paymentRepo.Setup(r => r.CreateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);

        var request = new CreatePaymentRequest
        {
            TenantId = "tenant-1",
            CustomerId = "cust-1",
            OrderId = "order-3",
            Amount = 75m,
            Currency = "USD",
            Method = "credit_card",
            PaymentMethodId = methodId
        };

        var result = await _service.ProcessPaymentAsync(request);

        result.Status.Should().Be("Completed");
        _gateway.Verify(g => g.ChargeAsync(It.Is<GatewayChargeRequest>(r => r.Token == "tok_stored_123"), It.IsAny<CancellationToken>()), Times.Once);
    }

    [Theory]
    [InlineData("credit_card", "CreditCard")]
    [InlineData("debit_card", "DebitCard")]
    [InlineData("bank_transfer", "BankTransfer")]
    [InlineData("digital_wallet", "DigitalWallet")]
    [InlineData("bkash", "bKash")]
    [InlineData("nagad", "Nagad")]
    [InlineData("rocket", "Rocket")]
    [InlineData("cod", "CashOnDelivery")]
    [InlineData("unknown_method", "bKash")] // defaults to bKash
    public async Task ProcessPayment_ShouldParsePaymentMethod(string input, string expected)
    {
        _gateway.Setup(g => g.ChargeAsync(It.IsAny<GatewayChargeRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse { Success = true, TransactionId = "gw_1" });
        _paymentRepo.Setup(r => r.CreateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);

        var request = new CreatePaymentRequest
        {
            TenantId = "t1", CustomerId = "c1", OrderId = $"o-{input}",
            Amount = 10m, Currency = "BDT", Method = input
        };

        var result = await _service.ProcessPaymentAsync(request);

        result.Method.Should().Be(expected);
    }

    #endregion

    #region GetPayment Tests

    [Fact]
    public async Task GetPaymentById_WhenExists_ShouldReturnPaymentDetail()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            TenantId = "tenant-1",
            OrderId = "order-1",
            CustomerId = "cust-1",
            Amount = 100m,
            Currency = "USD",
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard,
            Transactions = new List<PaymentTransaction>
            {
                new() { Id = Guid.NewGuid(), TenantId = "tenant-1", PaymentId = paymentId, Type = TransactionType.Charge, Status = TransactionStatus.Success, Amount = 100m, Currency = "USD", TransactionDate = DateTime.UtcNow }
            },
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(paymentId, It.IsAny<CancellationToken>()))
            .ReturnsAsync(payment);

        var result = await _service.GetPaymentByIdAsync(paymentId);

        result.Should().NotBeNull();
        result!.Id.Should().Be(paymentId);
        result.Transactions.Should().HaveCount(1);
        result.Refunds.Should().BeEmpty();
    }

    [Fact]
    public async Task GetPaymentById_WhenNotExists_ShouldReturnNull()
    {
        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment?)null);

        var result = await _service.GetPaymentByIdAsync(Guid.NewGuid());

        result.Should().BeNull();
    }

    [Fact]
    public async Task GetPaymentByOrderId_WhenExists_ShouldReturnPayment()
    {
        var payment = new Payment
        {
            Id = Guid.NewGuid(),
            TenantId = "tenant-1",
            OrderId = "order-99",
            CustomerId = "cust-1",
            Amount = 200m,
            Currency = "USD",
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard,
            Transactions = new List<PaymentTransaction>(),
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByOrderIdAsync("tenant-1", "order-99", It.IsAny<CancellationToken>()))
            .ReturnsAsync(payment);

        var result = await _service.GetPaymentByOrderIdAsync("tenant-1", "order-99");

        result.Should().NotBeNull();
        result!.OrderId.Should().Be("order-99");
    }

    [Fact]
    public async Task GetPaymentsPaged_ShouldReturnPagedResults()
    {
        var payments = new List<Payment>
        {
            new() { Id = Guid.NewGuid(), TenantId = "t1", OrderId = "o1", CustomerId = "c1", Amount = 10m, Currency = "USD", Status = PaymentStatus.Completed, Method = PaymentMethodType.CreditCard },
            new() { Id = Guid.NewGuid(), TenantId = "t1", OrderId = "o2", CustomerId = "c2", Amount = 20m, Currency = "USD", Status = PaymentStatus.Completed, Method = PaymentMethodType.CreditCard }
        };

        _paymentRepo.Setup(r => r.GetPagedAsync("t1", 0, 10, null, It.IsAny<CancellationToken>()))
            .ReturnsAsync((payments, 2));

        var (items, total) = await _service.GetPaymentsPagedAsync("t1", 0, 10);

        items.Should().HaveCount(2);
        total.Should().Be(2);
    }

    [Fact]
    public async Task GetPaymentsPaged_WithStatusFilter_ShouldPassFilter()
    {
        _paymentRepo.Setup(r => r.GetPagedAsync("t1", 0, 10, PaymentStatus.Failed, It.IsAny<CancellationToken>()))
            .ReturnsAsync((new List<Payment>(), 0));

        var (items, total) = await _service.GetPaymentsPagedAsync("t1", 0, 10, "Failed");

        items.Should().BeEmpty();
        total.Should().Be(0);
        _paymentRepo.Verify(r => r.GetPagedAsync("t1", 0, 10, PaymentStatus.Failed, It.IsAny<CancellationToken>()), Times.Once);
    }

    [Fact]
    public async Task GetPaymentsByCustomer_ShouldReturnCustomerPayments()
    {
        var payments = new List<Payment>
        {
            new() { Id = Guid.NewGuid(), TenantId = "t1", OrderId = "o1", CustomerId = "cust-5", Amount = 50m, Currency = "USD", Status = PaymentStatus.Completed, Method = PaymentMethodType.CreditCard }
        };

        _paymentRepo.Setup(r => r.GetByCustomerIdAsync("t1", "cust-5", It.IsAny<CancellationToken>()))
            .ReturnsAsync(payments);

        var result = await _service.GetPaymentsByCustomerAsync("t1", "cust-5");

        result.Should().HaveCount(1);
        result[0].CustomerId.Should().Be("cust-5");
    }

    #endregion

    #region CancelPayment Tests

    [Fact]
    public async Task CancelPayment_WhenPending_ShouldSucceed()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            TenantId = "t1",
            OrderId = "o1",
            CustomerId = "c1",
            Amount = 100m,
            Currency = "USD",
            Status = PaymentStatus.Pending,
            Method = PaymentMethodType.CreditCard
        };

        _paymentRepo.Setup(r => r.GetByIdAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);

        var result = await _service.CancelPaymentAsync(paymentId, new CancelPaymentRequest { Reason = "changed mind", CancelledBy = "user" });

        result.Status.Should().Be("Cancelled");
    }

    [Fact]
    public async Task CancelPayment_WhenCompleted_ShouldThrow()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard
        };

        _paymentRepo.Setup(r => r.GetByIdAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);

        var act = () => _service.CancelPaymentAsync(paymentId, new CancelPaymentRequest { Reason = "test" });

        await act.Should().ThrowAsync<InvalidOperationException>()
            .WithMessage("*cannot cancel*");
    }

    [Fact]
    public async Task CancelPayment_WithGatewayTransaction_ShouldVoid()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            TenantId = "t1",
            OrderId = "o1",
            CustomerId = "c1",
            Amount = 100m,
            Currency = "USD",
            Status = PaymentStatus.Processing,
            Method = PaymentMethodType.CreditCard,
            GatewayTransactionId = "gw_txn_to_void"
        };

        _paymentRepo.Setup(r => r.GetByIdAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _gateway.Setup(g => g.VoidAsync("gw_txn_to_void", It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse { Success = true, TransactionId = "gw_void_1" });
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);

        var result = await _service.CancelPaymentAsync(paymentId, new CancelPaymentRequest { Reason = "test", CancelledBy = "user" });

        result.Status.Should().Be("Cancelled");
        _gateway.Verify(g => g.VoidAsync("gw_txn_to_void", It.IsAny<CancellationToken>()), Times.Once);
        _transactionRepo.Verify(r => r.CreateAsync(It.Is<PaymentTransaction>(t => t.Type == TransactionType.Void), It.IsAny<CancellationToken>()), Times.Once);
    }

    [Fact]
    public async Task CancelPayment_WhenNotFound_ShouldThrowKeyNotFound()
    {
        _paymentRepo.Setup(r => r.GetByIdAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment?)null);

        var act = () => _service.CancelPaymentAsync(Guid.NewGuid(), new CancelPaymentRequest { Reason = "test" });

        await act.Should().ThrowAsync<KeyNotFoundException>();
    }

    #endregion

    #region Refund Tests

    [Fact]
    public async Task RefundPayment_FullRefund_ShouldSucceed()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            TenantId = "t1",
            OrderId = "o1",
            CustomerId = "c1",
            Amount = 100m,
            RefundedAmount = 0m,
            Currency = "USD",
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard,
            GatewayTransactionId = "gw_123",
            Transactions = new List<PaymentTransaction>(),
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _refundRepo.Setup(r => r.CreateAsync(It.IsAny<Refund>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Refund r, CancellationToken _) => r);
        _refundRepo.Setup(r => r.UpdateAsync(It.IsAny<Refund>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Refund r, CancellationToken _) => r);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);
        _gateway.Setup(g => g.RefundAsync(It.IsAny<GatewayRefundRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse { Success = true, TransactionId = "gw_rf_1" });

        var result = await _service.RefundPaymentAsync(paymentId, new RefundPaymentRequest { Reason = "Customer request" });

        result.Status.Should().Be("Completed");
        result.Amount.Should().Be(100m);
        result.GatewayRefundId.Should().Be("gw_rf_1");
    }

    [Fact]
    public async Task RefundPayment_PartialRefund_ShouldSetPartiallyRefunded()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            TenantId = "t1",
            OrderId = "o1",
            CustomerId = "c1",
            Amount = 100m,
            RefundedAmount = 0m,
            Currency = "USD",
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard,
            GatewayTransactionId = "gw_123",
            Transactions = new List<PaymentTransaction>(),
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _refundRepo.Setup(r => r.CreateAsync(It.IsAny<Refund>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Refund r, CancellationToken _) => r);
        _refundRepo.Setup(r => r.UpdateAsync(It.IsAny<Refund>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Refund r, CancellationToken _) => r);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);
        _gateway.Setup(g => g.RefundAsync(It.IsAny<GatewayRefundRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse { Success = true, TransactionId = "gw_rf_partial" });

        var result = await _service.RefundPaymentAsync(paymentId, new RefundPaymentRequest { Reason = "partial", Amount = 40m });

        result.Status.Should().Be("Completed");
        result.Amount.Should().Be(40m);
        // Payment status should be PartiallyRefunded (verified through the updated payment object)
        payment.Status.Should().Be(PaymentStatus.PartiallyRefunded);
        payment.RefundedAmount.Should().Be(40m);
    }

    [Fact]
    public async Task RefundPayment_ExceedsRefundableAmount_ShouldThrow()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            Amount = 100m,
            RefundedAmount = 80m,
            Status = PaymentStatus.PartiallyRefunded,
            Method = PaymentMethodType.CreditCard,
            GatewayTransactionId = "gw_123",
            Transactions = new List<PaymentTransaction>(),
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);

        var act = () => _service.RefundPaymentAsync(paymentId, new RefundPaymentRequest { Reason = "too much", Amount = 30m });

        await act.Should().ThrowAsync<InvalidOperationException>()
            .WithMessage("*exceeds remaining*");
    }

    [Fact]
    public async Task RefundPayment_WhenPending_ShouldThrow()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            Status = PaymentStatus.Pending,
            Method = PaymentMethodType.CreditCard,
            Transactions = new List<PaymentTransaction>(),
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);

        var act = () => _service.RefundPaymentAsync(paymentId, new RefundPaymentRequest { Reason = "test" });

        await act.Should().ThrowAsync<InvalidOperationException>()
            .WithMessage("*cannot refund*");
    }

    [Fact]
    public async Task RefundPayment_WhenNotFound_ShouldThrowKeyNotFound()
    {
        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment?)null);

        var act = () => _service.RefundPaymentAsync(Guid.NewGuid(), new RefundPaymentRequest { Reason = "test" });

        await act.Should().ThrowAsync<KeyNotFoundException>();
    }

    [Fact]
    public async Task RefundPayment_WhenGatewayFails_ShouldReturnFailedRefund()
    {
        var paymentId = Guid.NewGuid();
        var payment = new Payment
        {
            Id = paymentId,
            TenantId = "t1",
            Amount = 100m,
            RefundedAmount = 0m,
            Currency = "USD",
            Status = PaymentStatus.Completed,
            Method = PaymentMethodType.CreditCard,
            GatewayTransactionId = "gw_123",
            Transactions = new List<PaymentTransaction>(),
            Refunds = new List<Refund>()
        };

        _paymentRepo.Setup(r => r.GetByIdWithDetailsAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(payment);
        _paymentRepo.Setup(r => r.UpdateAsync(It.IsAny<Payment>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Payment p, CancellationToken _) => p);
        _refundRepo.Setup(r => r.CreateAsync(It.IsAny<Refund>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Refund r, CancellationToken _) => r);
        _refundRepo.Setup(r => r.UpdateAsync(It.IsAny<Refund>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((Refund r, CancellationToken _) => r);
        _transactionRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentTransaction>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentTransaction t, CancellationToken _) => t);
        _gateway.Setup(g => g.RefundAsync(It.IsAny<GatewayRefundRequest>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync(new GatewayResponse { Success = false, ErrorMessage = "refund_failed" });

        var result = await _service.RefundPaymentAsync(paymentId, new RefundPaymentRequest { Reason = "test" });

        result.Status.Should().Be("Failed");
        result.FailureReason.Should().Be("refund_failed");
    }

    [Fact]
    public async Task GetRefundById_WhenExists_ShouldReturn()
    {
        var refund = new Refund
        {
            Id = Guid.NewGuid(),
            TenantId = "t1",
            PaymentId = Guid.NewGuid(),
            Amount = 50m,
            Currency = "USD",
            Reason = "test",
            Status = RefundStatus.Completed
        };

        _refundRepo.Setup(r => r.GetByIdAsync(refund.Id, It.IsAny<CancellationToken>())).ReturnsAsync(refund);

        var result = await _service.GetRefundByIdAsync(refund.Id);

        result.Should().NotBeNull();
        result!.Amount.Should().Be(50m);
    }

    [Fact]
    public async Task GetRefundsByPayment_ShouldReturnList()
    {
        var paymentId = Guid.NewGuid();
        var refunds = new List<Refund>
        {
            new() { Id = Guid.NewGuid(), TenantId = "t1", PaymentId = paymentId, Amount = 30m, Currency = "USD", Reason = "r1", Status = RefundStatus.Completed },
            new() { Id = Guid.NewGuid(), TenantId = "t1", PaymentId = paymentId, Amount = 20m, Currency = "USD", Reason = "r2", Status = RefundStatus.Completed }
        };

        _refundRepo.Setup(r => r.GetByPaymentIdAsync(paymentId, It.IsAny<CancellationToken>())).ReturnsAsync(refunds);

        var result = await _service.GetRefundsByPaymentAsync(paymentId);

        result.Should().HaveCount(2);
    }

    #endregion
}
