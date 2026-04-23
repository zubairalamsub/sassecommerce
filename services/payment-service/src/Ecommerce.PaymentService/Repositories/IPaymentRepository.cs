using Ecommerce.PaymentService.Entities;

namespace Ecommerce.PaymentService.Repositories;

public interface IPaymentRepository
{
    Task<Payment?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<Payment?> GetByIdWithDetailsAsync(Guid id, CancellationToken cancellationToken = default);
    Task<Payment?> GetByOrderIdAsync(string tenantId, string orderId, CancellationToken cancellationToken = default);
    Task<Payment?> GetByIdempotencyKeyAsync(string idempotencyKey, CancellationToken cancellationToken = default);
    Task<List<Payment>> GetByCustomerIdAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task<(List<Payment> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, PaymentStatus? status = null, CancellationToken cancellationToken = default);
    Task<Payment> CreateAsync(Payment payment, CancellationToken cancellationToken = default);
    Task<Payment> UpdateAsync(Payment payment, CancellationToken cancellationToken = default);
}
