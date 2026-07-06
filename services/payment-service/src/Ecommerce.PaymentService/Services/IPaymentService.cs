using Ecommerce.PaymentService.DTOs;

namespace Ecommerce.PaymentService.Services;

public interface IPaymentService
{
    // Payment operations
    Task<PaymentResponse> ProcessPaymentAsync(CreatePaymentRequest request, CancellationToken cancellationToken = default);
    Task<PaymentResponse?> CompleteGatewayPaymentAsync(string gatewayTransactionId, string bankTransactionId, decimal amount, string rawResponse, CancellationToken cancellationToken = default);
    Task<PaymentResponse?> FailGatewayPaymentAsync(string gatewayTransactionId, string reason, string rawResponse, CancellationToken cancellationToken = default);
    Task<PaymentDetailResponse?> GetPaymentByIdAsync(Guid id, string tenantId, CancellationToken cancellationToken = default);
    Task<PaymentDetailResponse?> GetPaymentByOrderIdAsync(string tenantId, string orderId, CancellationToken cancellationToken = default);
    Task<(List<PaymentResponse> Items, int Total)> GetPaymentsPagedAsync(string tenantId, int offset, int limit, string? status = null, CancellationToken cancellationToken = default);
    Task<List<PaymentResponse>> GetPaymentsByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task<PaymentResponse> CancelPaymentAsync(Guid id, string tenantId, CancelPaymentRequest request, CancellationToken cancellationToken = default);

    // Refund operations
    Task<RefundResponse> RefundPaymentAsync(Guid paymentId, string tenantId, RefundPaymentRequest request, CancellationToken cancellationToken = default);
    Task<RefundResponse?> GetRefundByIdAsync(Guid refundId, string tenantId, CancellationToken cancellationToken = default);
    Task<List<RefundResponse>> GetRefundsByPaymentAsync(Guid paymentId, string tenantId, CancellationToken cancellationToken = default);

    // Payment method operations
    Task<PaymentMethodResponse> CreatePaymentMethodAsync(CreatePaymentMethodRequest request, CancellationToken cancellationToken = default);
    Task<PaymentMethodResponse> UpdatePaymentMethodAsync(Guid id, string tenantId, UpdatePaymentMethodRequest request, CancellationToken cancellationToken = default);
    Task<PaymentMethodResponse?> GetPaymentMethodByIdAsync(Guid id, string tenantId, CancellationToken cancellationToken = default);
    Task<List<PaymentMethodResponse>> GetPaymentMethodsByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task DeletePaymentMethodAsync(Guid id, string tenantId, CancellationToken cancellationToken = default);
}
