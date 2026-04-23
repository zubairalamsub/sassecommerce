using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.PaymentService.Repositories;

public class PaymentMethodRepository : IPaymentMethodRepository
{
    private readonly PaymentDbContext _context;

    public PaymentMethodRepository(PaymentDbContext context)
    {
        _context = context;
    }

    public async Task<PaymentMethod?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.PaymentMethods
            .FirstOrDefaultAsync(pm => pm.Id == id, cancellationToken);
    }

    public async Task<List<PaymentMethod>> GetByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default)
    {
        return await _context.PaymentMethods
            .Where(pm => pm.TenantId == tenantId && pm.CustomerId == customerId && pm.IsActive)
            .OrderByDescending(pm => pm.IsDefault)
            .ThenByDescending(pm => pm.CreatedAt)
            .ToListAsync(cancellationToken);
    }

    public async Task<PaymentMethod?> GetDefaultAsync(string tenantId, string customerId, CancellationToken cancellationToken = default)
    {
        return await _context.PaymentMethods
            .FirstOrDefaultAsync(pm =>
                pm.TenantId == tenantId &&
                pm.CustomerId == customerId &&
                pm.IsDefault &&
                pm.IsActive,
                cancellationToken);
    }

    public async Task<PaymentMethod> CreateAsync(PaymentMethod paymentMethod, CancellationToken cancellationToken = default)
    {
        _context.PaymentMethods.Add(paymentMethod);
        await _context.SaveChangesAsync(cancellationToken);
        return paymentMethod;
    }

    public async Task<PaymentMethod> UpdateAsync(PaymentMethod paymentMethod, CancellationToken cancellationToken = default)
    {
        _context.PaymentMethods.Update(paymentMethod);
        await _context.SaveChangesAsync(cancellationToken);
        return paymentMethod;
    }

    public async Task DeleteAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var method = await GetByIdAsync(id, cancellationToken);
        if (method != null)
        {
            method.DeletedAt = DateTime.UtcNow;
            method.IsActive = false;
            await _context.SaveChangesAsync(cancellationToken);
        }
    }

    public async Task ClearDefaultAsync(string tenantId, string customerId, CancellationToken cancellationToken = default)
    {
        var defaults = await _context.PaymentMethods
            .Where(pm => pm.TenantId == tenantId && pm.CustomerId == customerId && pm.IsDefault)
            .ToListAsync(cancellationToken);

        foreach (var pm in defaults)
        {
            pm.IsDefault = false;
        }

        await _context.SaveChangesAsync(cancellationToken);
    }
}
