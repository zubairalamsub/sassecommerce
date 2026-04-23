using Ecommerce.PaymentService.Entities;

namespace Ecommerce.PaymentService.Repositories;

public interface IRefundRepository
{
    Task<Refund?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default);
    Task<List<Refund>> GetByPaymentIdAsync(Guid paymentId, CancellationToken cancellationToken = default);
    Task<Refund> CreateAsync(Refund refund, CancellationToken cancellationToken = default);
    Task<Refund> UpdateAsync(Refund refund, CancellationToken cancellationToken = default);
}
