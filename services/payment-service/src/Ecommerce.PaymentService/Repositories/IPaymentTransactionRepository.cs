using Ecommerce.PaymentService.Entities;

namespace Ecommerce.PaymentService.Repositories;

public interface IPaymentTransactionRepository
{
    Task<PaymentTransaction?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<PaymentTransaction>> GetByPaymentIdAsync(Guid paymentId, CancellationToken cancellationToken = default);
    Task<PaymentTransaction> CreateAsync(PaymentTransaction transaction, CancellationToken cancellationToken = default);
    Task<PaymentTransaction> UpdateAsync(PaymentTransaction transaction, CancellationToken cancellationToken = default);
}
