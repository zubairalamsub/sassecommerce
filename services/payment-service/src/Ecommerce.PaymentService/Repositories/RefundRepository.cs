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

    public async Task<Refund?> GetByIdAsync(Guid id, string tenantId, CancellationToken cancellationToken = default)
    {
        return await _context.Refunds
            .Include(r => r.Payment)
            .FirstOrDefaultAsync(r => r.Id == id && r.TenantId == tenantId, cancellationToken);
    }

    public async Task<List<Refund>> GetByPaymentIdAsync(Guid paymentId, string tenantId, CancellationToken cancellationToken = default)
    {
        return await _context.Refunds
            .Where(r => r.PaymentId == paymentId && r.TenantId == tenantId)
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
