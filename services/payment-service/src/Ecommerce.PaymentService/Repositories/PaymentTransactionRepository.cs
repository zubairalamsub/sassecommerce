using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.PaymentService.Repositories;

public class PaymentTransactionRepository : IPaymentTransactionRepository
{
    private readonly PaymentDbContext _context;

    public PaymentTransactionRepository(PaymentDbContext context)
    {
        _context = context;
    }

    public async Task<PaymentTransaction?> GetByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        return await _context.PaymentTransactions
            .Include(t => t.Payment)
            .FirstOrDefaultAsync(t => t.Id == id, cancellationToken);
    }

    public async Task<List<PaymentTransaction>> GetByPaymentIdAsync(Guid paymentId, CancellationToken cancellationToken = default)
    {
        return await _context.PaymentTransactions
            .Where(t => t.PaymentId == paymentId)
            .OrderByDescending(t => t.TransactionDate)
            .ToListAsync(cancellationToken);
    }

    public async Task<PaymentTransaction> CreateAsync(PaymentTransaction transaction, CancellationToken cancellationToken = default)
    {
        _context.PaymentTransactions.Add(transaction);
        await _context.SaveChangesAsync(cancellationToken);
        return transaction;
    }

    public async Task<PaymentTransaction> UpdateAsync(PaymentTransaction transaction, CancellationToken cancellationToken = default)
    {
        _context.PaymentTransactions.Update(transaction);
        await _context.SaveChangesAsync(cancellationToken);
        return transaction;
    }
}
