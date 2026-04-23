using AutoMapper;
using Ecommerce.InventoryService.DTOs;
using Ecommerce.InventoryService.Entities;

namespace Ecommerce.InventoryService.Mappings;

public class MappingProfile : Profile
{
    public MappingProfile()
    {
        // Warehouse mappings
        CreateMap<CreateWarehouseRequest, Warehouse>();
        CreateMap<Warehouse, WarehouseResponse>();

        // InventoryItem mappings
        CreateMap<CreateInventoryItemRequest, InventoryItem>()
            .ForMember(dest => dest.QuantityOnHand, opt => opt.Ignore())
            .ForMember(dest => dest.QuantityReserved, opt => opt.Ignore());

        // StockReservation mappings
        CreateMap<StockReservation, StockReservationResponse>()
            .ForMember(dest => dest.Status, opt => opt.MapFrom(src => src.Status.ToString()));
    }
}
