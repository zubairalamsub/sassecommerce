using AutoMapper;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Entities;

namespace Ecommerce.PaymentService.Mappings;

public class MappingProfile : Profile
{
    public MappingProfile()
    {
        // Payment mappings
        CreateMap<Payment, PaymentResponse>()
            .ForMember(dest => dest.Status, opt => opt.MapFrom(src => src.Status.ToString()))
            .ForMember(dest => dest.Method, opt => opt.MapFrom(src => src.Method.ToString()));

        CreateMap<Payment, PaymentDetailResponse>()
            .ForMember(dest => dest.Status, opt => opt.MapFrom(src => src.Status.ToString()))
            .ForMember(dest => dest.Method, opt => opt.MapFrom(src => src.Method.ToString()))
            .ForMember(dest => dest.Transactions, opt => opt.Ignore())
            .ForMember(dest => dest.Refunds, opt => opt.Ignore());

        // PaymentTransaction mappings
        CreateMap<PaymentTransaction, PaymentTransactionResponse>()
            .ForMember(dest => dest.Type, opt => opt.MapFrom(src => src.Type.ToString()))
            .ForMember(dest => dest.Status, opt => opt.MapFrom(src => src.Status.ToString()));

        // Refund mappings
        CreateMap<Refund, RefundResponse>()
            .ForMember(dest => dest.Status, opt => opt.MapFrom(src => src.Status.ToString()));

        // PaymentMethod mappings
        CreateMap<PaymentMethod, PaymentMethodResponse>()
            .ForMember(dest => dest.Type, opt => opt.MapFrom(src => src.Type.ToString()));
    }
}
