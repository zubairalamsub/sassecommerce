using Ecommerce.InventoryService.Entities;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.InventoryService.Data;

public class InventoryDbContext : DbContext
{
    public InventoryDbContext(DbContextOptions<InventoryDbContext> options) : base(options)
    {
    }

    public DbSet<Warehouse> Warehouses { get; set; } = null!;
    public DbSet<InventoryItem> InventoryItems { get; set; } = null!;
    public DbSet<StockMovement> StockMovements { get; set; } = null!;
    public DbSet<StockReservation> StockReservations { get; set; } = null!;

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);

        // Warehouse configuration
        modelBuilder.Entity<Warehouse>(entity =>
        {
            entity.ToTable("warehouses");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => new { e.TenantId, e.Code }).IsUnique();
            entity.HasIndex(e => e.TenantId);
            entity.Property(e => e.Code).HasMaxLength(50).IsRequired();
            entity.Property(e => e.Name).HasMaxLength(200).IsRequired();
            entity.Property(e => e.Address).HasMaxLength(500).IsRequired();
            entity.Property(e => e.City).HasMaxLength(100).IsRequired();
            entity.Property(e => e.State).HasMaxLength(100).IsRequired();
            entity.Property(e => e.Country).HasMaxLength(100).IsRequired();
            entity.Property(e => e.PostalCode).HasMaxLength(20).IsRequired();
            entity.Property(e => e.Phone).HasMaxLength(20);
            entity.Property(e => e.Email).HasMaxLength(200);

            // Soft delete filter
            entity.HasQueryFilter(e => e.DeletedAt == null);
        });

        // InventoryItem configuration
        modelBuilder.Entity<InventoryItem>(entity =>
        {
            entity.ToTable("inventory_items");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => new { e.TenantId, e.WarehouseId, e.ProductId, e.VariantId }).IsUnique();
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.SKU);
            entity.HasIndex(e => e.ProductId);
            entity.Property(e => e.SKU).HasMaxLength(100).IsRequired();
            entity.Property(e => e.ProductId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.VariantId).HasMaxLength(100);
            entity.Property(e => e.BinLocation).HasMaxLength(50);
            entity.Ignore(e => e.QuantityAvailable); // Computed property

            entity.HasOne(e => e.Warehouse)
                .WithMany(w => w.InventoryItems)
                .HasForeignKey(e => e.WarehouseId)
                .OnDelete(DeleteBehavior.Restrict);

            // Soft delete filter
            entity.HasQueryFilter(e => e.DeletedAt == null);
        });

        // StockMovement configuration
        modelBuilder.Entity<StockMovement>(entity =>
        {
            entity.ToTable("stock_movements");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.InventoryItemId);
            entity.HasIndex(e => e.MovementDate);
            entity.HasIndex(e => e.OrderId);
            entity.Property(e => e.Reference).HasMaxLength(100);
            entity.Property(e => e.OrderId).HasMaxLength(100);
            entity.Property(e => e.Reason).HasMaxLength(500);

            entity.HasOne(e => e.InventoryItem)
                .WithMany(i => i.StockMovements)
                .HasForeignKey(e => e.InventoryItemId)
                .OnDelete(DeleteBehavior.Restrict);

            entity.HasOne(e => e.Warehouse)
                .WithMany(w => w.StockMovements)
                .HasForeignKey(e => e.WarehouseId)
                .OnDelete(DeleteBehavior.Restrict);

            // Soft delete filter
            entity.HasQueryFilter(e => e.DeletedAt == null);
        });

        // StockReservation configuration
        modelBuilder.Entity<StockReservation>(entity =>
        {
            entity.ToTable("stock_reservations");
            entity.HasKey(e => e.Id);
            entity.HasIndex(e => e.TenantId);
            entity.HasIndex(e => e.InventoryItemId);
            entity.HasIndex(e => e.OrderId);
            entity.HasIndex(e => e.Status);
            entity.HasIndex(e => e.ExpiresAt);
            entity.Property(e => e.OrderId).HasMaxLength(100).IsRequired();
            entity.Property(e => e.OrderItemId).HasMaxLength(100).IsRequired();

            entity.HasOne(e => e.InventoryItem)
                .WithMany(i => i.StockReservations)
                .HasForeignKey(e => e.InventoryItemId)
                .OnDelete(DeleteBehavior.Restrict);

            // Soft delete filter
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
