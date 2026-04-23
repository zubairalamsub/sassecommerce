using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.PaymentService.Repositories;

public class RefundRepository : IRefundRepository
{
    private readonly PaymentDbContext _context;

    public RefundRepository(PaymentDbContext context)
    {
        _context = context;
    }

    public async Task<Refund?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.Refunds
            .Include(r => r.Payment)
            .FirstOrDefaultAsync(r => r.Id == id, cancellationToken);
    }

    public async Task<List<Refund>> GetByPaymentIdAsync(Guid paymentId, CancellationToken cancellationToken = default)
    {
        return await _context.Refunds
            .Where(r => r.PaymentId == paymentId)
            .OrderByDescending(r => r.CreatedAt)
            .ToListAsync(cancellationToken);
    }

    public async Task<Refund> CreateAsync(Refund refund, CancellationToken cancellationToken = default)
    {
        _context.Refunds.Add(refund);
        await _context.SaveChangesAsync(cancellationToken);
        return refund;
    }

    public async Task<Refund> UpdateAsync(Refund refund, CancellationToken cancellationToken = default)
    {
        _context.Refunds.Update(refund);
        await _context.SaveChangesAsync(cancellationToken);
        return refund;
    }
}
