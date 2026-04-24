using Ecommerce.PaymentService.DTOs;

namespace Ecommerce.PaymentService.Services;

public interface IPaymentService
{
    // Payment operations
    Task<PaymentResponse> ProcessPaymentAsync(CreatePaymentRequest request, CancellationToken cancellationToken = default);
    Task<PaymentResponse?> CompleteGatewayPaymentAsync(string gatewayTransactionId, string bankTransactionId, decimal amount, string rawResponse, CancellationToken cancellationToken = default);
    Task<PaymentResponse?> FailGatewayPaymentAsync(string gatewayTransactionId, string reason, string rawResponse, CancellationToken cancellationToken = default);
    Task<PaymentDetailResponse?> GetPaymentByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<PaymentDetailResponse?> GetPaymentByOrderIdAsync(string tenantId, string orderId, CancellationToken cancellationToken = default);
    Task<(List<PaymentResponse> Items, int Total)> GetPaymentsPagedAsync(string tenantId, int offset, int limit, string? status = null, CancellationToken cancellationToken = default);
    Task<List<PaymentResponse>> GetPaymentsByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task<PaymentResponse> CancelPaymentAsync(Guid id, CancelPaymentRequest request, CancellationToken cancellationToken = default);

    // Refund operations
    Task<RefundResponse> RefundPaymentAsync(Guid paymentId, RefundPaymentRequest request, CancellationToken cancellationToken = default);
    Task<RefundResponse?> GetRefundByIdAsync(Guid refundId, CancellationToken cancellationToken = default);
    Task<List<RefundResponse>> GetRefundsByPaymentAsync(Guid paymentId, CancellationToken cancellationToken = default);

    // Payment method operations
    Task<PaymentMethodResponse> CreatePaymentMethodAsync(CreatePaymentMethodRequest request, CancellationToken cancellationToken = default);
    Task<PaymentMethodResponse> UpdatePaymentMethodAsync(Guid id, UpdatePaymentMethodRequest request, CancellationToken cancellationToken = default);
    Task<PaymentMethodResponse?> GetPaymentMethodByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<PaymentMethodResponse>> GetPaymentMethodsByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task DeletePaymentMethodAsync(Guid id, CancellationToken cancellationToken = default);
}
