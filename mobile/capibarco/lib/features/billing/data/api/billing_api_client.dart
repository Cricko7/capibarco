import '../../../../core/network/rest_service_client.dart';
import '../dtos/donation_intent_dto.dart';

class BillingApiClient {
  const BillingApiClient(this._client);

  final RestServiceClient _client;

  Future<DonationIntentDto> createDonationIntent({
    required String targetType,
    required String targetId,
    required int units,
    required int nanos,
    required String currencyCode,
    String provider = 'mock',
  }) async {
    final response = await _client.postJson(
      '/billing/donation-intents',
      idempotent: true,
      data: <String, dynamic>{
        'target_type': targetType,
        'target_id': targetId,
        'amount': <String, dynamic>{
          'currency_code': currencyCode,
          'units': units,
          'nanos': nanos,
        },
        'provider': provider,
      },
    );
    return DonationIntentDto.fromJson(response);
  }
}
