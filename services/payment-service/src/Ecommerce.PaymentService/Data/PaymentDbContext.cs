using Ecommerce.PaymentService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.PaymentService.Data;

public class PaymentDbContext : DbContext
{
    public PaymentDbContext(DbContextOptions<PaymentDbContext> options) : base(options)
    {
    }

    public DbSet<Payment> Payments { get; set; } = null!;
    public DbSet<PaymentMethod> PaymentMethods { get; set; } = null!;
    public DbSet<PaymentTransaction> PaymentTransactions { get; set; } = null!;
    public DbSet<Refund> Refunds { get; set; } = null!;

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);

        // Payment configuration
        modelBuilder.Entity<Payment>(entity =>
        {
            entity.ToTable("payments");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.OrderId);
            entity.HasIndex(e => e.CustomerId);
            entity.HasIndex(e => e.Status);
            entity.HasIndex(e => new { e.TenantId, e.OrderId });
            entity.HasIndex(e => e.IdempotencyKey).IsUnique().HasFilter("idempotency_key IS NOT NULL");

            entity.Property(e => e.TenantId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.OrderId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.CustomerId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.Amount).HasPrecision(18, 2);
            entity.Property(e => e.RefundedAmount).HasPrecision(18, 2);
            entity.Property(e => e.Currency).HasMaxLength(3).IsRequired();
            entity.Property(e => e.Status).HasConversion<string>().HasMaxLength(30);
            entity.Property(e => e.Method).HasConversion<string>().HasMaxLength(30);
            entity.Property(e => e.GatewayName).HasMaxLength(50);
            entity.Property(e => e.GatewayTransactionId).HasMaxLength(200);
            entity.Property(e => e.GatewayResponse).HasMaxLength(2000);
            entity.Property(e => e.FailureReason).HasMaxLength(500);
            entity.Property(e => e.Description).HasMaxLength(500);
            entity.Property(e => e.IdempotencyKey).HasMaxLength(100);

            entity.HasOne(e => e.PaymentMethodNavigation)
                .WithMany(pm => pm.Payments)
                .HasForeignKey(e => e.PaymentMethodId)
                .OnDelete(DeleteBehavior.SetNull);

            entity.HasQueryFilter(e => e.DeletedAt == null);
        });

        // PaymentMethod configuration
        modelBuilder.Entity<PaymentMethod>(entity =>
        {
            entity.ToTable("payment_methods");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.CustomerId);
            entity.HasIndex(e => new { e.TenantId, e.CustomerId });

            entity.Property(e => e.TenantId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.CustomerId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.Type).HasConversion<string>().HasMaxLength(30);
            entity.Property(e => e.Token).HasMaxLength(500);
            entity.Property(e => e.Last4).HasMaxLength(4);
            entity.Property(e => e.Brand).HasMaxLength(50);
            entity.Property(e => e.CardholderName).HasMaxLength(200);
            entity.Property(e => e.BankName).HasMaxLength(200);
            entity.Property(e => e.AccountLast4).HasMaxLength(4);
            entity.Property(e => e.WalletProvider).HasMaxLength(50);
            entity.Property(e => e.WalletEmail).HasMaxLength(200);
            entity.Property(e => e.BillingAddressLine1).HasMaxLength(200);
            entity.Property(e => e.BillingAddressLine2).HasMaxLength(200);
            entity.Property(e => e.BillingCity).HasMaxLength(100);
            entity.Property(e => e.BillingState).HasMaxLength(100);
            entity.Property(e => e.BillingPostalCode).HasMaxLength(20);
            entity.Property(e => e.BillingCountry).HasMaxLength(100);

            entity.HasQueryFilter(e => e.DeletedAt == null);
        });

        // PaymentTransaction configuration
        modelBuilder.Entity<PaymentTransaction>(entity =>
        {
            entity.ToTable("payment_transactions");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.PaymentId);
            entity.HasIndex(e => e.Type);
            entity.HasIndex(e => e.TransactionDate);

            entity.Property(e => e.TenantId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.Amount).HasPrecision(18, 2);
            entity.Property(e => e.Currency).HasMaxLength(3).IsRequired();
            entity.Property(e => e.Type).HasConversion<string>().HasMaxLength(30);
            entity.Property(e => e.Status).HasConversion<string>().HasMaxLength(30);
            entity.Property(e => e.GatewayTransactionId).HasMaxLength(200);
            entity.Property(e => e.GatewayResponse).HasMaxLength(2000);
            entity.Property(e => e.GatewayErrorCode).HasMaxLength(50);
            entity.Property(e => e.GatewayErrorMessage).HasMaxLength(500);
            entity.Property(e => e.Reference).HasMaxLength(100);
            entity.Property(e => e.Notes).HasMaxLength(500);

            entity.HasOne(e => e.Payment)
                .WithMany(p => p.Transactions)
                .HasForeignKey(e => e.PaymentId)
                .OnDelete(DeleteBehavior.Restrict);

            entity.HasQueryFilter(e => e.DeletedAt == null);
        });

        // Refund configuration
        modelBuilder.Entity<Refund>(entity =>
        {
            entity.ToTable("refunds");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.PaymentId);
            entity.HasIndex(e => e.Status);

            entity.Property(e => e.TenantId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.Amount).HasPrecision(18, 2);
            entity.Property(e => e.Currency).HasMaxLength(3).IsRequired();
            entity.Property(e => e.Reason).HasMaxLength(500).IsRequired();
            entity.Property(e => e.Status).HasConversion<string>().HasMaxLength(30);
            entity.Property(e => e.FailureReason).HasMaxLength(500);
            entity.Property(e => e.GatewayRefundId).HasMaxLength(200);
            entity.Property(e => e.GatewayResponse).HasMaxLength(2000);

            entity.HasOne(e => e.Payment)
                .WithMany(p => p.Refunds)
                .HasForeignKey(e => e.PaymentId)
                .OnDelete(DeleteBehavior.Restrict);

            entity.HasQueryFilter(e => e.DeletedAt == null);
        });
    }

    public override int SaveChanges()
    {
        UpdateTimestamps();
        return base.SaveChanges();
    }

    public override Task<int> SaveChangesAsync(CancellationToken cancellationToken = default)
    {
        UpdateTimestamps();
        return base.SaveChangesAsync(cancellationToken);
    }

    private void UpdateTimestamps()
    {
        var entries = ChangeTracker.Entries()
            .Where(e => e.Entity is BaseEntity && (e.State == EntityState.Added || e.State == EntityState.Modified));

        foreach (var entry in entries)
        {
            var entity = (BaseEntity)entry.Entity;

            if (entry.State == EntityState.Added)
            {
                entity.CreatedAt = DateTime.UtcNow;
                entity.UpdatedAt = DateTime.UtcNow;
            }
            else if (entry.State == EntityState.Modified)
            {
                entity.UpdatedAt = DateTime.UtcNow;
            }
        }
    }
}
