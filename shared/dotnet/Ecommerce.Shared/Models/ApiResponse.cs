namespace Ecommerce.Shared.Models;

public class ApiResponse<T>
{
    public bool Success { get; set; }
    public T? Data { get; set; }
    public string? Message { get; set; }

    public static ApiResponse<T> SuccessResponse(T data, string? message = null)
    {
        return new ApiResponse<T>
        {
            Success = true,
            Data = data,
            Message = message
        };
    }
}

public class ErrorResponse
{
    public bool Success { get; set; } = false;
    public string Error { get; set; } = string.Empty;
    public string? Code { get; set; }
    public object? Details { get; set; }
    public string? RequestId { get; set; }
}

public class PaginatedResponse<T>
{
    public bool Success { get; set; }
    public IEnumerable<T> Data { get; set; } = Array.Empty<T>();
    public PaginationMetadata Pagination { get; set; } = new();

    public static PaginatedResponse<T> Create(
        IEnumerable<T> data,
        int page,
        int pageSize,
        long totalItems)
    {
        return new PaginatedResponse<T>
        {
            Success = true,
            Data = data,
            Pagination = new PaginationMetadata
            {
                Page = page,
                PageSize = pageSize,
                TotalItems = totalItems,
                TotalPages = (int)Math.Ceiling(totalItems / (double)pageSize)
            }
        };
    }
}

public class PaginationMetadata
{
    public int Page { get; set; }
    public int PageSize { get; set; }
    public long TotalItems { get; set; }
    public int TotalPages { get; set; }
}
