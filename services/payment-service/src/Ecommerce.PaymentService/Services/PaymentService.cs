using AutoMapper;
using Ecommerce.PaymentService.DTOs;
using Ecommerce.PaymentService.Entities;
using Ecommerce.PaymentService.Messaging;
using Ecommerce.PaymentService.Repositories;

namespace Ecommerce.PaymentService.Services;

public class PaymentService : IPaymentService
{
    private readonly IPaymentRepository _paymentRepository;
    private readonly IPaymentMethodRepository _paymentMethodRepository;
    private readonly IPaymentTransactionRepository _transactionRepository;
    private readonly IRefundRepository _refundRepository;
    private readonly IPaymentGateway _gateway;
    private readonly IEventPublisher _eventPublisher;
    private readonly IMapper _mapper;
    private readonly ILogger<PaymentService> _logger;

    public PaymentService(
        IPaymentRepository paymentRepository,
        IPaymentMethodRepository paymentMethodRepository,
        IPaymentTransactionRepository transactionRepository,
        IRefundRepository refundRepository,
        IPaymentGateway gateway,
        IEventPublisher eventPublisher,
        IMapper mapper,
        ILogger<PaymentService> logger)
    {
        _paymentRepository = paymentRepository;
        _paymentMethodRepository = paymentMethodRepository;
        _transactionRepository = transactionRepository;
        _refundRepository = refundRepository;
        _gateway = gateway;
        _eventPublisher = eventPublisher;
        _mapper = mapper;
        _logger = logger;
    }

    #region Payment Operations

    public async Task<PaymentResponse> ProcessPaymentAsync(CreatePaymentRequest request, CancellationToken cancellationToken = default)
    {
        // Idempotency check
        if (!string.IsNullOrEmpty(request.IdempotencyKey))
        {
            var existing = await _paymentRepository.GetByIdempotencyKeyAsync(request.IdempotencyKey, cancellationToken);
            if (existing != null)
            {
                _logger.LogInformation("Returning existing payment for idempotency key: {Key}", request.IdempotencyKey);
                return _mapper.Map<PaymentResponse>(existing);
            }
        }

        // Parse payment method type
        var methodType = ParsePaymentMethodType(request.Method);

        // Create payment record
        var payment = new Payment
        {
            Id = Guid.NewGuid(),
            TenantId = request.TenantId,
            OrderId = request.OrderId,
            CustomerId = request.CustomerId,
            Amount = request.Amount,
            Currency = request.Currency,
            Method = methodType,
            PaymentMethodId = request.PaymentMethodId,
            Status = PaymentStatus.Processing,
            GatewayName = _gateway.Name,
            Description = request.Description,
            IdempotencyKey = request.IdempotencyKey,
            ProcessedAt = DateTime.UtcNow,
            CreatedBy = request.CreatedBy
        };

        await _paymentRepository.CreateAsync(payment, cancellationToken);

        _logger.LogInformation("Payment created: {PaymentId} for Order {OrderId}, Amount: {Amount} {Currency}",
            payment.Id, request.OrderId, request.Amount, request.Currency);

        // Get token from stored payment method if provided
        string? token = null;
        if (request.PaymentMethodId.HasValue)
        {
            var paymentMethod = await _paymentMethodRepository.GetByIdAsync(request.PaymentMethodId.Value, cancellationToken);
            token = paymentMethod?.Token;
        }

        // Process via gateway
        var chargeRequest = new GatewayChargeRequest
        {
            Amount = request.Amount,
            Currency = request.Currency,
            Token = token,
            Description = request.Description,
            CustomerId = request.CustomerId,
            OrderId = request.OrderId,
            Metadata = new Dictionary<string, string>
            {
                { "tenant_id", request.TenantId },
                { "payment_id", payment.Id.ToString() }
            }
        };

        var gatewayResponse = await _gateway.ChargeAsync(chargeRequest, cancellationToken);

        // Record transaction
        var transaction = new PaymentTransaction
        {
            Id = Guid.NewGuid(),
            TenantId = request.TenantId,
            PaymentId = payment.Id,
            Type = TransactionType.Charge,
            Amount = request.Amount,
            Currency = request.Currency,
            GatewayTransactionId = gatewayResponse.TransactionId,
            GatewayResponse = gatewayResponse.RawResponse,
            GatewayErrorCode = gatewayResponse.ErrorCode,
            GatewayErrorMessage = gatewayResponse.ErrorMessage,
            TransactionDate = DateTime.UtcNow,
            CreatedBy = request.CreatedBy
        };

        if (gatewayResponse.Success)
        {
            payment.Status = PaymentStatus.Completed;
            payment.GatewayTransactionId = gatewayResponse.TransactionId;
            payment.GatewayResponse = gatewayResponse.RawResponse;
            payment.CompletedAt = DateTime.UtcNow;
            transaction.Status = TransactionStatus.Success;

            _logger.LogInformation("Payment completed: {PaymentId}, GatewayTxn: {GatewayTxnId}",
                payment.Id, gatewayResponse.TransactionId);
        }
        else
        {
            payment.Status = PaymentStatus.Failed;
            payment.FailureReason = gatewayResponse.ErrorMessage;
            payment.GatewayResponse = gatewayResponse.RawResponse;
            payment.FailedAt = DateTime.UtcNow;
            transaction.Status = TransactionStatus.Failed;

            _logger.LogWarning("Payment failed: {PaymentId}, Error: {Error}",
                payment.Id, gatewayResponse.ErrorMessage);
        }

        await _paymentRepository.UpdateAsync(payment, cancellationToken);
        await _transactionRepository.CreateAsync(transaction, cancellationToken);

        // Publish payment event
        if (gatewayResponse.Success)
        {
            await _eventPublisher.PublishAsync("PaymentCompleted", new Dictionary<string, object>
            {
                ["tenant_id"] = payment.TenantId,
                ["payment_id"] = payment.Id.ToString(),
                ["order_id"] = payment.OrderId,
                ["customer_id"] = payment.CustomerId,
                ["amount"] = payment.Amount,
                ["currency"] = payment.Currency
            }, cancellationToken);
        }
        else
        {
            await _eventPublisher.PublishAsync("PaymentFailed", new Dictionary<string, object>
            {
                ["tenant_id"] = payment.TenantId,
                ["payment_id"] = payment.Id.ToString(),
                ["order_id"] = payment.OrderId,
                ["customer_id"] = payment.CustomerId,
                ["amount"] = payment.Amount,
                ["reason"] = payment.FailureReason ?? "Unknown"
            }, cancellationToken);
        }

        return _mapper.Map<PaymentResponse>(payment);
    }

    public async Task<PaymentDetailResponse?> GetPaymentByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var payment = await _paymentRepository.GetByIdWithDetailsAsync(id, cancellationToken);
        return payment == null ? null : MapToDetailResponse(payment);
    }

    public async Task<PaymentDetailResponse?> GetPaymentByOrderIdAsync(string tenantId, string orderId, CancellationToken cancellationToken = default)
    {
        var payment = await _paymentRepository.GetByOrderIdAsync(tenantId, orderId, cancellationToken);
        return payment == null ? null : MapToDetailResponse(payment);
    }

    public async Task<(List<PaymentResponse> Items, int Total)> GetPaymentsPagedAsync(string tenantId, int offset, int limit, string? status = null, CancellationToken cancellationToken = default)
    {
        PaymentStatus? statusFilter = null;
        if (!string.IsNullOrEmpty(status) && Enum.TryParse<PaymentStatus>(status, true, out var parsed))
        {
            statusFilter = parsed;
        }

        var (payments, total) = await _paymentRepository.GetPagedAsync(tenantId, offset, limit, statusFilter, cancellationToken);
        return (_mapper.Map<List<PaymentResponse>>(payments), total);
    }

    public async Task<List<PaymentResponse>> GetPaymentsByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default)
    {
        var payments = await _paymentRepository.GetByCustomerIdAsync(tenantId, customerId, cancellationToken);
        return _mapper.Map<List<PaymentResponse>>(payments);
    }

    public async Task<PaymentResponse> CancelPaymentAsync(Guid id, CancelPaymentRequest request, CancellationToken cancellationToken = default)
    {
        var payment = await _paymentRepository.GetByIdAsync(id, cancellationToken)
            ?? throw new KeyNotFoundException($"Payment {id} not found");

        if (payment.Status != PaymentStatus.Pending && payment.Status != PaymentStatus.Processing)
        {
            throw new InvalidOperationException($"Payment is {payment.Status}, cannot cancel. Only Pending or Processing payments can be cancelled.");
        }

        // If there's a gateway transaction, void it
        if (!string.IsNullOrEmpty(payment.GatewayTransactionId))
        {
            var voidResponse = await _gateway.VoidAsync(payment.GatewayTransactionId, cancellationToken);

            var voidTransaction = new PaymentTransaction
            {
                Id = Guid.NewGuid(),
                TenantId = payment.TenantId,
                PaymentId = payment.Id,
                Type = TransactionType.Void,
                Amount = payment.Amount,
                Currency = payment.Currency,
                GatewayTransactionId = voidResponse.TransactionId,
                GatewayResponse = voidResponse.RawResponse,
                Status = voidResponse.Success ? TransactionStatus.Success : TransactionStatus.Failed,
                TransactionDate = DateTime.UtcNow,
                Notes = request.Reason,
                CreatedBy = request.CancelledBy
            };

            await _transactionRepository.CreateAsync(voidTransaction, cancellationToken);
        }

        payment.Status = PaymentStatus.Cancelled;
        payment.CancelledAt = DateTime.UtcNow;
        payment.UpdatedBy = request.CancelledBy;

        await _paymentRepository.UpdateAsync(payment, cancellationToken);

        _logger.LogInformation("Payment cancelled: {PaymentId}, Reason: {Reason}", id, request.Reason);

        return _mapper.Map<PaymentResponse>(payment);
    }

    #endregion

    #region Refund Operations

    public async Task<RefundResponse> RefundPaymentAsync(Guid paymentId, RefundPaymentRequest request, CancellationToken cancellationToken = default)
    {
        var payment = await _paymentRepository.GetByIdWithDetailsAsync(paymentId, cancellationToken)
            ?? throw new KeyNotFoundException($"Payment {paymentId} not found");

        if (payment.Status != PaymentStatus.Completed && payment.Status != PaymentStatus.PartiallyRefunded)
        {
            throw new InvalidOperationException($"Payment is {payment.Status}, cannot refund. Only Completed or PartiallyRefunded payments can be refunded.");
        }

        // Determine refund amount (full or partial)
        var refundAmount = request.Amount ?? (payment.Amount - payment.RefundedAmount);

        if (refundAmount <= 0)
        {
            throw new InvalidOperationException("Refund amount must be greater than zero");
        }

        var remainingRefundable = payment.Amount - payment.RefundedAmount;
        if (refundAmount > remainingRefundable)
        {
            throw new InvalidOperationException($"Refund amount ({refundAmount}) exceeds remaining refundable amount ({remainingRefundable})");
        }

        // Create refund record
        var refund = new Refund
        {
            Id = Guid.NewGuid(),
            TenantId = payment.TenantId,
            PaymentId = paymentId,
            Amount = refundAmount,
            Currency = payment.Currency,
            Reason = request.Reason,
            Status = RefundStatus.Processing,
            ProcessedAt = DateTime.UtcNow,
            CreatedBy = request.CreatedBy
        };

        await _refundRepository.CreateAsync(refund, cancellationToken);

        // Process via gateway
        var gatewayRefundRequest = new GatewayRefundRequest
        {
            TransactionId = payment.GatewayTransactionId!,
            Amount = refundAmount,
            Currency = payment.Currency,
            Reason = request.Reason
        };

        var gatewayResponse = await _gateway.RefundAsync(gatewayRefundRequest, cancellationToken);

        // Record transaction
        var transaction = new PaymentTransaction
        {
            Id = Guid.NewGuid(),
            TenantId = payment.TenantId,
            PaymentId = paymentId,
            Type = TransactionType.Refund,
            Amount = refundAmount,
            Currency = payment.Currency,
            GatewayTransactionId = gatewayResponse.TransactionId,
            GatewayResponse = gatewayResponse.RawResponse,
            GatewayErrorCode = gatewayResponse.ErrorCode,
            GatewayErrorMessage = gatewayResponse.ErrorMessage,
            TransactionDate = DateTime.UtcNow,
            Notes = request.Reason,
            CreatedBy = request.CreatedBy
        };

        if (gatewayResponse.Success)
        {
            refund.Status = RefundStatus.Completed;
            refund.GatewayRefundId = gatewayResponse.TransactionId;
            refund.GatewayResponse = gatewayResponse.RawResponse;
            refund.CompletedAt = DateTime.UtcNow;
            transaction.Status = TransactionStatus.Success;

            // Update payment
            payment.RefundedAmount += refundAmount;
            if (payment.RefundedAmount >= payment.Amount)
            {
                payment.Status = PaymentStatus.Refunded;
            }
            else
            {
                payment.Status = PaymentStatus.PartiallyRefunded;
            }

            _logger.LogInformation("Refund completed: {RefundId} for Payment {PaymentId}, Amount: {Amount}",
                refund.Id, paymentId, refundAmount);
        }
        else
        {
            refund.Status = RefundStatus.Failed;
            refund.FailureReason = gatewayResponse.ErrorMessage;
            refund.FailedAt = DateTime.UtcNow;
            transaction.Status = TransactionStatus.Failed;

            _logger.LogWarning("Refund failed: {RefundId}, Error: {Error}",
                refund.Id, gatewayResponse.ErrorMessage);
        }

        await _refundRepository.UpdateAsync(refund, cancellationToken);
        await _transactionRepository.CreateAsync(transaction, cancellationToken);
        await _paymentRepository.UpdateAsync(payment, cancellationToken);

        return _mapper.Map<RefundResponse>(refund);
    }

    public async Task<RefundResponse?> GetRefundByIdAsync(Guid refundId, CancellationToken cancellationToken = default)
    {
        var refund = await _refundRepository.GetByIdAsync(refundId, cancellationToken);
        return refund == null ? null : _mapper.Map<RefundResponse>(refund);
    }

    public async Task<List<RefundResponse>> GetRefundsByPaymentAsync(Guid paymentId, CancellationToken cancellationToken = default)
    {
        var refunds = await _refundRepository.GetByPaymentIdAsync(paymentId, cancellationToken);
        return _mapper.Map<List<RefundResponse>>(refunds);
    }

    #endregion

    #region Payment Method Operations

    public async Task<PaymentMethodResponse> CreatePaymentMethodAsync(CreatePaymentMethodRequest request, CancellationToken cancellationToken = default)
    {
        var methodType = ParsePaymentMethodType(request.Type);

        var paymentMethod = new PaymentMethod
        {
            Id = Guid.NewGuid(),
            TenantId = request.TenantId,
            CustomerId = request.CustomerId,
            Type = methodType,
            IsDefault = request.IsDefault,
            BillingAddressLine1 = request.BillingAddressLine1,
            BillingAddressLine2 = request.BillingAddressLine2,
            BillingCity = request.BillingCity,
            BillingState = request.BillingState,
            BillingPostalCode = request.BillingPostalCode,
            BillingCountry = request.BillingCountry,
            CreatedBy = request.CreatedBy
        };

        // Tokenize and store based on type
        switch (methodType)
        {
            case PaymentMethodType.CreditCard:
            case PaymentMethodType.DebitCard:
                if (!string.IsNullOrEmpty(request.CardNumber))
                {
                    paymentMethod.Token = await _gateway.TokenizeCardAsync(
                        request.CardNumber, request.ExpiryMonth ?? 0, request.ExpiryYear ?? 0, "", cancellationToken);
                    paymentMethod.Last4 = request.CardNumber.Length >= 4
                        ? request.CardNumber[^4..]
                        : request.CardNumber;
                    paymentMethod.Brand = DetectCardBrand(request.CardNumber);
                }
                paymentMethod.ExpiryMonth = request.ExpiryMonth;
                paymentMethod.ExpiryYear = request.ExpiryYear;
                paymentMethod.CardholderName = request.CardholderName;
                break;

            case PaymentMethodType.BankTransfer:
                paymentMethod.BankName = request.BankName;
                if (!string.IsNullOrEmpty(request.AccountNumber))
                {
                    paymentMethod.AccountLast4 = request.AccountNumber.Length >= 4
                        ? request.AccountNumber[^4..]
                        : request.AccountNumber;
                    paymentMethod.Token = $"bank_{Guid.NewGuid():N}";
                }
                break;

            case PaymentMethodType.DigitalWallet:
                paymentMethod.WalletProvider = request.WalletProvider;
                paymentMethod.WalletEmail = request.WalletEmail;
                paymentMethod.Token = $"wallet_{Guid.NewGuid():N}";
                break;

            case PaymentMethodType.bKash:
            case PaymentMethodType.Nagad:
            case PaymentMethodType.Rocket:
                paymentMethod.WalletProvider = methodType.ToString();
                paymentMethod.WalletEmail = request.WalletEmail; // stores mobile number
                paymentMethod.Token = $"mfs_{Guid.NewGuid():N}";
                break;

            case PaymentMethodType.CashOnDelivery:
                paymentMethod.Token = $"cod_{Guid.NewGuid():N}";
                break;
        }

        // If setting as default, clear other defaults
        if (request.IsDefault)
        {
            await _paymentMethodRepository.ClearDefaultAsync(request.TenantId, request.CustomerId, cancellationToken);
        }

        await _paymentMethodRepository.CreateAsync(paymentMethod, cancellationToken);

        _logger.LogInformation("Payment method created: {PaymentMethodId} for Customer {CustomerId}, Type: {Type}",
            paymentMethod.Id, request.CustomerId, methodType);

        return _mapper.Map<PaymentMethodResponse>(paymentMethod);
    }

    public async Task<PaymentMethodResponse> UpdatePaymentMethodAsync(Guid id, UpdatePaymentMethodRequest request, CancellationToken cancellationToken = default)
    {
        var paymentMethod = await _paymentMethodRepository.GetByIdAsync(id, cancellationToken)
            ?? throw new KeyNotFoundException($"Payment method {id} not found");

        if (request.ExpiryMonth.HasValue) paymentMethod.ExpiryMonth = request.ExpiryMonth;
        if (request.ExpiryYear.HasValue) paymentMethod.ExpiryYear = request.ExpiryYear;
        if (request.CardholderName != null) paymentMethod.CardholderName = request.CardholderName;
        if (request.BillingAddressLine1 != null) paymentMethod.BillingAddressLine1 = request.BillingAddressLine1;
        if (request.BillingAddressLine2 != null) paymentMethod.BillingAddressLine2 = request.BillingAddressLine2;
        if (request.BillingCity != null) paymentMethod.BillingCity = request.BillingCity;
        if (request.BillingState != null) paymentMethod.BillingState = request.BillingState;
        if (request.BillingPostalCode != null) paymentMethod.BillingPostalCode = request.BillingPostalCode;
        if (request.BillingCountry != null) paymentMethod.BillingCountry = request.BillingCountry;

        if (request.IsDefault.HasValue && request.IsDefault.Value)
        {
            await _paymentMethodRepository.ClearDefaultAsync(paymentMethod.TenantId, paymentMethod.CustomerId, cancellationToken);
            paymentMethod.IsDefault = true;
        }
        else if (request.IsDefault.HasValue)
        {
            paymentMethod.IsDefault = false;
        }

        paymentMethod.UpdatedBy = request.UpdatedBy;

        await _paymentMethodRepository.UpdateAsync(paymentMethod, cancellationToken);

        _logger.LogInformation("Payment method updated: {PaymentMethodId}", id);

        return _mapper.Map<PaymentMethodResponse>(paymentMethod);
    }

    public async Task<PaymentMethodResponse?> GetPaymentMethodByIdAsync(Guid id, CancellationToken cancellationToken = default)
    {
        var paymentMethod = await _paymentMethodRepository.GetByIdAsync(id, cancellationToken);
        return paymentMethod == null ? null : _mapper.Map<PaymentMethodResponse>(paymentMethod);
    }

    public async Task<List<PaymentMethodResponse>> GetPaymentMethodsByCustomerAsync(string tenantId, string customerId, CancellationToken cancellationToken = default)
    {
        var methods = await _paymentMethodRepository.GetByCustomerAsync(tenantId, customerId, cancellationToken);
        return _mapper.Map<List<PaymentMethodResponse>>(methods);
    }

    public async Task DeletePaymentMethodAsync(Guid id, CancellationToken cancellationToken = default)
    {
        await _paymentMethodRepository.DeleteAsync(id, cancellationToken);
        _logger.LogInformation("Payment method deleted: {PaymentMethodId}", id);
    }

    #endregion

    #region Helper Methods

    private PaymentDetailResponse MapToDetailResponse(Payment payment)
    {
        var response = _mapper.Map<PaymentDetailResponse>(payment);
        response.Transactions = _mapper.Map<List<PaymentTransactionResponse>>(payment.Transactions);
        response.Refunds = _mapper.Map<List<RefundResponse>>(payment.Refunds);
        return response;
    }

    private static PaymentMethodType ParsePaymentMethodType(string method)
    {
        return method.ToLowerInvariant().Replace("_", "").Replace("-", "") switch
        {
            "creditcard" => PaymentMethodType.CreditCard,
            "debitcard" => PaymentMethodType.DebitCard,
            "banktransfer" => PaymentMethodType.BankTransfer,
            "digitalwallet" => PaymentMethodType.DigitalWallet,
            "bkash" => PaymentMethodType.bKash,
            "nagad" => PaymentMethodType.Nagad,
            "rocket" => PaymentMethodType.Rocket,
            "cashondelivery" or "cod" => PaymentMethodType.CashOnDelivery,
            _ => PaymentMethodType.bKash
        };
    }

    private static string DetectCardBrand(string cardNumber)
    {
        if (string.IsNullOrEmpty(cardNumber)) return "Unknown";

        return cardNumber[0] switch
        {
            '4' => "Visa",
            '5' => "Mastercard",
            '3' when cardNumber.Length >= 2 && (cardNumber[1] == '4' || cardNumber[1] == '7') => "Amex",
            '6' => "Discover",
            _ => "Unknown"
        };
    }

    #endregion
}
