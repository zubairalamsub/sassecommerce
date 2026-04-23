using AutoMapper;
using Ecommerce.PaymentService.Data;
using Ecommerce.PaymentService.Mappings;
using Microsoft.EntityFrameworkCore;

namespace Ecommerce.PaymentService.Tests;

public static class TestHelpers
{
    public static PaymentDbContext CreateInMemoryDbContext(string? dbName = null)
    {
        var options = new DbContextOptionsBuilder<PaymentDbContext>()
            .UseInMemoryDatabase(databaseName: dbName ?? Guid.NewGuid().ToString())
            .Options;

        return new PaymentDbContext(options);
    }

    public static IMapper CreateMapper()
    {
        var config = new MapperConfiguration(cfg =>
        {
            cfg.AddProfile<MappingProfile>();
        });

        return config.CreateMapper();
    }
}
