using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.PaymentService.Repositories;

public class PaymentRepository : IPaymentRepository
{
    private readonly PaymentDbContext _context;

    public PaymentRepository(PaymentDbContext context)
    {
        _context = context;
    }

    public async Task<Payment?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.Payments
            .FirstOrDefaultAsync(p => p.Id == id, cancellationToken);
    }

    public async Task<Payment?> GetByIdWithDetailsAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.Payments
            .Include(p => p.Transactions.OrderByDescending(t => t.TransactionDate))
            .Include(p => p.Refunds.OrderByDescending(r => r.CreatedAt))
            .Include(p => p.PaymentMethodNavigation)
            .FirstOrDefaultAsync(p => p.Id == id, cancellationToken);
    }

    public async Task<Payment?> GetByOrderIdAsync(string tenantId, string orderId, CancellationToken cancellationToken = default)
    {
        return await _context.Payments
            .Include(p => p.Transactions.OrderByDescending(t => t.TransactionDate))
            .Include(p => p.Refunds.OrderByDescending(r => r.CreatedAt))
            .FirstOrDefaultAsync(p => p.TenantId == tenantId && p.OrderId == orderId, cancellationToken);
    }

    public async Task<Payment?> GetByIdempotencyKeyAsync(string idempotencyKey, CancellationToken cancellationToken = default)
    {
        return await _context.Payments
            .FirstOrDefaultAsync(p => p.IdempotencyKey == idempotencyKey, cancellationToken);
    }

    public async Task<Payment?> GetByGatewayTransactionIdAsync(string gatewayTransactionId, CancellationToken cancellationToken = default)
    {
        return await _context.Payments
            .FirstOrDefaultAsync(p => p.GatewayTransactionId == gatewayTransactionId, cancellationToken);
    }

    public async Task<List<Payment>> GetByCustomerIdAsync(string tenantId, string customerId, CancellationToken cancellationToken = default)
    {
        return await _context.Payments
            .Where(p => p.TenantId == tenantId && p.CustomerId == customerId)
            .OrderByDescending(p => p.CreatedAt)
            .ToListAsync(cancellationToken);
    }

    public async Task<(List<Payment> Items, int Total)> GetPagedAsync(string tenantId, int offset, int limit, PaymentStatus? status = null, CancellationToken cancellationToken = default)
    {
        var query = _context.Payments
            .Where(p => p.TenantId == tenantId);

        if (status.HasValue)
        {
            query = query.Where(p => p.Status == status.Value);
        }

        var total = await query.CountAsync(cancellationToken);
        var items = await query
            .OrderByDescending(p => p.CreatedAt)
            .Skip(offset)
            .Take(limit)
            .ToListAsync(cancellationToken);

        return (items, total);
    }

    public async Task<Payment> CreateAsync(Payment payment, CancellationToken cancellationToken = default)
    {
        _context.Payments.Add(payment);
        await _context.SaveChangesAsync(cancellationToken);
        return payment;
    }

    public async Task<Payment> UpdateAsync(Payment payment, CancellationToken cancellationToken = default)
    {
        _context.Payments.Update(payment);
        await _context.SaveChangesAsync(cancellationToken);
        return payment;
    }
}
