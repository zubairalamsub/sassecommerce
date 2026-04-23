using Ecommerce.PaymentService.Services;
using FluentAssertions;
using Microsoft.Extensions.Logging;
using Moq;

namespace Ecommerce.PaymentService.Tests;

public class SimulatedPaymentGatewayTests
{
    private readonly SimulatedPaymentGateway _gateway;

    public SimulatedPaymentGatewayTests()
    {
        var logger = new Mock<ILogger<SimulatedPaymentGateway>>();
        _gateway = new SimulatedPaymentGateway(logger.Object);
    }

    [Fact]
    public void Name_ShouldReturnSimulatedGateway()
    {
        _gateway.Name.Should().Be("SimulatedGateway");
    }

    [Fact]
    public async Task ChargeAsync_WithValidAmount_ShouldSucceed()
    {
        var request = new GatewayChargeRequest
        {
            Amount = 99.99m,
            Currency = "USD",
            OrderId = "ORD-001",
            CustomerId = "CUST-001"
        };

        var result = await _gateway.ChargeAsync(request);

        result.Success.Should().BeTrue();
        result.TransactionId.Should().StartWith("sim_ch_");
        result.ErrorCode.Should().BeNull();
        result.ErrorMessage.Should().BeNull();
        result.RawResponse.Should().NotBeNullOrEmpty();
    }

    [Fact]
    public async Task ChargeAsync_WithZeroAmount_ShouldFail()
    {
        var request = new GatewayChargeRequest
        {
            Amount = 0,
            Currency = "USD"
        };

        var result = await _gateway.ChargeAsync(request);

        result.Success.Should().BeFalse();
        result.ErrorCode.Should().Be("invalid_amount");
        result.ErrorMessage.Should().Contain("greater than zero");
    }

    [Fact]
    public async Task RefundAsync_WithValidTransaction_ShouldSucceed()
    {
        var request = new GatewayRefundRequest
        {
            TransactionId = "sim_ch_abc123",
            Amount = 50.00m,
            Currency = "USD",
            Reason = "Customer request"
        };

        var result = await _gateway.RefundAsync(request);

        result.Success.Should().BeTrue();
        result.TransactionId.Should().StartWith("sim_rf_");
        result.RawResponse.Should().NotBeNullOrEmpty();
    }

    [Fact]
    public async Task RefundAsync_WithEmptyTransactionId_ShouldFail()
    {
        var request = new GatewayRefundRequest
        {
            TransactionId = "",
            Amount = 50.00m
        };

        var result = await _gateway.RefundAsync(request);

        result.Success.Should().BeFalse();
        result.ErrorCode.Should().Be("invalid_transaction");
    }

    [Fact]
    public async Task VoidAsync_ShouldSucceed()
    {
        var result = await _gateway.VoidAsync("sim_ch_abc123");

        result.Success.Should().BeTrue();
        result.TransactionId.Should().StartWith("sim_vo_");
    }

    [Fact]
    public async Task TokenizeCardAsync_ShouldReturnToken()
    {
        var token = await _gateway.TokenizeCardAsync("4111111111111111", 12, 2027, "123");

        token.Should().StartWith("tok_");
        token.Length.Should().BeGreaterThan(4);
    }

    [Fact]
    public async Task TokenizeCardAsync_WithTestCard_ShouldStillReturnToken()
    {
        // Test card that triggers "card_declined" behavior
        var token = await _gateway.TokenizeCardAsync("4000000000000002", 12, 2027, "123");

        token.Should().StartWith("tok_");
    }

    [Fact]
    public async Task ChargeAsync_MultipleCalls_ShouldReturnUniqueTransactionIds()
    {
        var request = new GatewayChargeRequest { Amount = 10.00m, Currency = "USD" };

        var result1 = await _gateway.ChargeAsync(request);
        var result2 = await _gateway.ChargeAsync(request);

        result1.TransactionId.Should().NotBe(result2.TransactionId);
    }
}
