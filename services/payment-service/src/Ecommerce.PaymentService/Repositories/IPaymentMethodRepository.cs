using Ecommerce.PaymentService.Entities;

namespace Ecommerce.PaymentService.Repositories;

public interface IPaymentMethodRepository
{
    Task<PaymentMethod?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<PaymentMethod>> GetByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task<PaymentMethod?> GetDefaultAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
    Task<PaymentMethod> CreateAsync(PaymentMethod paymentMethod, CancellationToken cancellationToken = default);
    Task<PaymentMethod> UpdateAsync(PaymentMethod paymentMethod, CancellationToken cancellationToken = default);
    Task DeleteAsync(Guid id, CancellationToken cancellationToken = default);
    Task ClearDefaultAsync(string tenantId, string customerId, CancellationToken cancellationToken = default);
}
