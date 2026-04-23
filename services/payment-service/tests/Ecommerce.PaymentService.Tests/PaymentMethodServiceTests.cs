using AutoMapper;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Entities;
using Ecommerce.PaymentService.Repositories;
using Ecommerce.PaymentService.Services;
using FluentAssertions;
using Microsoft.Extensions.Logging;
using Moq;

namespace Ecommerce.PaymentService.Tests;

public class PaymentMethodServiceTests
{
    private readonly Mock<IPaymentRepository> _paymentRepo;
    private readonly Mock<IPaymentMethodRepository> _methodRepo;
    private readonly Mock<IPaymentTransactionRepository> _transactionRepo;
    private readonly Mock<IRefundRepository> _refundRepo;
    private readonly Mock<IPaymentGateway> _gateway;
    private readonly IMapper _mapper;
    private readonly Services.PaymentService _service;

    public PaymentMethodServiceTests()
    {
        _paymentRepo = new Mock<IPaymentRepository>();
        _methodRepo = new Mock<IPaymentMethodRepository>();
        _transactionRepo = new Mock<IPaymentTransactionRepository>();
        _refundRepo = new Mock<IRefundRepository>();
        _gateway = new Mock<IPaymentGateway>();
        _mapper = TestHelpers.CreateMapper();

        _gateway.Setup(g => g.Name).Returns("TestGateway");
        _gateway.Setup(g => g.TokenizeCardAsync(It.IsAny<string>(), It.IsAny<int>(), It.IsAny<int>(), It.IsAny<string>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync("tok_test_123");

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

    [Fact]
    public async Task CreatePaymentMethod_CreditCard_ShouldTokenizeAndDetectBrand()
    {
        _methodRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);

        var request = new CreatePaymentMethodRequest
        {
            TenantId = "t1",
            CustomerId = "c1",
            Type = "credit_card",
            CardNumber = "4111111111111111",
            ExpiryMonth = 12,
            ExpiryYear = 2027,
            CardholderName = "John Doe",
            IsDefault = false,
            CreatedBy = "test"
        };

        var result = await _service.CreatePaymentMethodAsync(request);

        result.Should().NotBeNull();
        result.Type.Should().Be("CreditCard");
        result.Last4.Should().Be("1111");
        result.Brand.Should().Be("Visa");
        result.ExpiryMonth.Should().Be(12);
        result.ExpiryYear.Should().Be(2027);
        result.CardholderName.Should().Be("John Doe");
        _gateway.Verify(g => g.TokenizeCardAsync("4111111111111111", 12, 2027, "", It.IsAny<CancellationToken>()), Times.Once);
    }

    [Theory]
    [InlineData("4111111111111111", "Visa")]
    [InlineData("5500000000000004", "Mastercard")]
    [InlineData("340000000000009", "Amex")]
    [InlineData("371449635398431", "Amex")]
    [InlineData("6011000000000004", "Discover")]
    [InlineData("9999999999999999", "Unknown")]
    public async Task CreatePaymentMethod_ShouldDetectCardBrand(string cardNumber, string expectedBrand)
    {
        _methodRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);

        var request = new CreatePaymentMethodRequest
        {
            TenantId = "t1",
            CustomerId = "c1",
            Type = "credit_card",
            CardNumber = cardNumber,
            ExpiryMonth = 12,
            ExpiryYear = 2027,
            CreatedBy = "test"
        };

        var result = await _service.CreatePaymentMethodAsync(request);

        result.Brand.Should().Be(expectedBrand);
    }

    [Fact]
    public async Task CreatePaymentMethod_AsDefault_ShouldClearOtherDefaults()
    {
        _methodRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);
        _methodRepo.Setup(r => r.ClearDefaultAsync("t1", "c1", It.IsAny<CancellationToken>()))
            .Returns(Task.CompletedTask);

        var request = new CreatePaymentMethodRequest
        {
            TenantId = "t1",
            CustomerId = "c1",
            Type = "credit_card",
            CardNumber = "4111111111111111",
            ExpiryMonth = 12,
            ExpiryYear = 2027,
            IsDefault = true,
            CreatedBy = "test"
        };

        var result = await _service.CreatePaymentMethodAsync(request);

        result.IsDefault.Should().BeTrue();
        _methodRepo.Verify(r => r.ClearDefaultAsync("t1", "c1", It.IsAny<CancellationToken>()), Times.Once);
    }

    [Fact]
    public async Task CreatePaymentMethod_BankTransfer_ShouldStoreBankDetails()
    {
        _methodRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);

        var request = new CreatePaymentMethodRequest
        {
            TenantId = "t1",
            CustomerId = "c1",
            Type = "bank_transfer",
            BankName = "Chase",
            AccountNumber = "123456789",
            CreatedBy = "test"
        };

        var result = await _service.CreatePaymentMethodAsync(request);

        result.Type.Should().Be("BankTransfer");
        result.BankName.Should().Be("Chase");
        result.AccountLast4.Should().Be("6789");
    }

    [Fact]
    public async Task CreatePaymentMethod_DigitalWallet_ShouldStoreWalletDetails()
    {
        _methodRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);

        var request = new CreatePaymentMethodRequest
        {
            TenantId = "t1",
            CustomerId = "c1",
            Type = "digital_wallet",
            WalletProvider = "Apple Pay",
            WalletEmail = "user@test.com",
            CreatedBy = "test"
        };

        var result = await _service.CreatePaymentMethodAsync(request);

        result.Type.Should().Be("DigitalWallet");
        result.WalletProvider.Should().Be("Apple Pay");
        result.WalletEmail.Should().Be("user@test.com");
    }

    [Fact]
    public async Task CreatePaymentMethod_bKash_ShouldSetTypeAutomatically()
    {
        _methodRepo.Setup(r => r.CreateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);

        var request = new CreatePaymentMethodRequest
        {
            TenantId = "t1",
            CustomerId = "c1",
            Type = "bkash",
            WalletEmail = "user@test.com",
            CreatedBy = "test"
        };

        var result = await _service.CreatePaymentMethodAsync(request);

        result.Type.Should().Be("bKash");
    }

    [Fact]
    public async Task UpdatePaymentMethod_ShouldUpdateFields()
    {
        var methodId = Guid.NewGuid();
        var existing = new PaymentMethod
        {
            Id = methodId,
            TenantId = "t1",
            CustomerId = "c1",
            Type = PaymentMethodType.CreditCard,
            ExpiryMonth = 12,
            ExpiryYear = 2025,
            CardholderName = "Old Name",
            BillingCity = "Old City"
        };

        _methodRepo.Setup(r => r.GetByIdAsync(methodId, It.IsAny<CancellationToken>())).ReturnsAsync(existing);
        _methodRepo.Setup(r => r.UpdateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);

        var request = new UpdatePaymentMethodRequest
        {
            ExpiryMonth = 6,
            ExpiryYear = 2028,
            CardholderName = "New Name",
            BillingCity = "New City",
            UpdatedBy = "admin"
        };

        var result = await _service.UpdatePaymentMethodAsync(methodId, request);

        result.ExpiryMonth.Should().Be(6);
        result.ExpiryYear.Should().Be(2028);
        result.CardholderName.Should().Be("New Name");
        result.BillingCity.Should().Be("New City");
    }

    [Fact]
    public async Task UpdatePaymentMethod_SetAsDefault_ShouldClearOthers()
    {
        var methodId = Guid.NewGuid();
        var existing = new PaymentMethod
        {
            Id = methodId,
            TenantId = "t1",
            CustomerId = "c1",
            Type = PaymentMethodType.CreditCard,
            IsDefault = false
        };

        _methodRepo.Setup(r => r.GetByIdAsync(methodId, It.IsAny<CancellationToken>())).ReturnsAsync(existing);
        _methodRepo.Setup(r => r.UpdateAsync(It.IsAny<PaymentMethod>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod pm, CancellationToken _) => pm);
        _methodRepo.Setup(r => r.ClearDefaultAsync("t1", "c1", It.IsAny<CancellationToken>()))
            .Returns(Task.CompletedTask);

        var result = await _service.UpdatePaymentMethodAsync(methodId, new UpdatePaymentMethodRequest { IsDefault = true, UpdatedBy = "test" });

        result.IsDefault.Should().BeTrue();
        _methodRepo.Verify(r => r.ClearDefaultAsync("t1", "c1", It.IsAny<CancellationToken>()), Times.Once);
    }

    [Fact]
    public async Task UpdatePaymentMethod_WhenNotFound_ShouldThrow()
    {
        _methodRepo.Setup(r => r.GetByIdAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod?)null);

        var act = () => _service.UpdatePaymentMethodAsync(Guid.NewGuid(), new UpdatePaymentMethodRequest { UpdatedBy = "test" });

        await act.Should().ThrowAsync<KeyNotFoundException>();
    }

    [Fact]
    public async Task GetPaymentMethodById_WhenExists_ShouldReturn()
    {
        var method = new PaymentMethod
        {
            Id = Guid.NewGuid(),
            TenantId = "t1",
            CustomerId = "c1",
            Type = PaymentMethodType.CreditCard,
            Last4 = "4242",
            Brand = "Visa"
        };

        _methodRepo.Setup(r => r.GetByIdAsync(method.Id, It.IsAny<CancellationToken>())).ReturnsAsync(method);

        var result = await _service.GetPaymentMethodByIdAsync(method.Id);

        result.Should().NotBeNull();
        result!.Last4.Should().Be("4242");
    }

    [Fact]
    public async Task GetPaymentMethodById_WhenNotExists_ShouldReturnNull()
    {
        _methodRepo.Setup(r => r.GetByIdAsync(It.IsAny<Guid>(), It.IsAny<CancellationToken>()))
            .ReturnsAsync((PaymentMethod?)null);

        var result = await _service.GetPaymentMethodByIdAsync(Guid.NewGuid());

        result.Should().BeNull();
    }

    [Fact]
    public async Task GetPaymentMethodsByCustomer_ShouldReturnList()
    {
        var methods = new List<PaymentMethod>
        {
            new() { Id = Guid.NewGuid(), TenantId = "t1", CustomerId = "c1", Type = PaymentMethodType.CreditCard },
            new() { Id = Guid.NewGuid(), TenantId = "t1", CustomerId = "c1", Type = PaymentMethodType.bKash }
        };

        _methodRepo.Setup(r => r.GetByCustomerAsync("t1", "c1", It.IsAny<CancellationToken>())).ReturnsAsync(methods);

        var result = await _service.GetPaymentMethodsByCustomerAsync("t1", "c1");

        result.Should().HaveCount(2);
    }

    [Fact]
    public async Task DeletePaymentMethod_ShouldCallRepository()
    {
        var methodId = Guid.NewGuid();
        _methodRepo.Setup(r => r.DeleteAsync(methodId, It.IsAny<CancellationToken>())).Returns(Task.CompletedTask);

        await _service.DeletePaymentMethodAsync(methodId);

        _methodRepo.Verify(r => r.DeleteAsync(methodId, It.IsAny<CancellationToken>()), Times.Once);
    }
}
