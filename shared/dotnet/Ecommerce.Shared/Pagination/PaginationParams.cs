using Microsoft.AspNetCore.Http;

namespace Ecommerce.Shared.Pagination;

public class PaginationParams
{
    private const int DefaultPage = 1;
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 100;

    public int Page { get; set; } = DefaultPage;
    public int PageSize { get; set; } = DefaultPageSize;
    public int Offset => (Page - 1) * PageSize;
    public string SortBy { get; set; } = "CreatedAt";
    public string SortDirection { get; set; } = "desc";

    public static PaginationParams FromQuery(HttpRequest request)
    {
        var page = GetIntQueryParam(request, "page", DefaultPage);
        if (page < 1) page = DefaultPage;

        var pageSize = GetIntQueryParam(request, "page_size", DefaultPageSize);
        if (pageSize < 1) pageSize = DefaultPageSize;
        if (pageSize > MaxPageSize) pageSize = MaxPageSize;

        var sortBy = request.Query["sort_by"].FirstOrDefault() ?? "CreatedAt";
        var sortDirection = request.Query["sort_dir"].FirstOrDefault() ?? "desc";

        if (sortDirection != "asc" && sortDirection != "desc")
        {
            sortDirection = "desc";
        }

        return new PaginationParams
        {
            Page = page,
            PageSize = pageSize,
            SortBy = sortBy,
            SortDirection = sortDirection
        };
    }

    public string GetOrderByClause()
    {
        var direction = SortDirection.Equals("asc", StringComparison.OrdinalIgnoreCase) ? "ASC" : "DESC";
        return $"{SortBy} {direction}";
    }

    private static int GetIntQueryParam(HttpRequest request, string key, int defaultValue)
    {
        var value = request.Query[key].FirstOrDefault();
        if (string.IsNullOrEmpty(value))
        {
            return defaultValue;
        }

        if (int.TryParse(value, out var result))
        {
            return result;
        }

        return defaultValue;
    }
}

public static class PaginationExtensions
{
    public static int CalculateTotalPages(long totalItems, int pageSize)
    {
        if (pageSize == 0) return 0;
        return (int)Math.Ceiling(totalItems / (double)pageSize);
    }
}
