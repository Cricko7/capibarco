import '../api/billing_api_client.dart';
import '../dtos/donation_intent_dto.dart';

class BillingRemoteDataSource {
  const BillingRemoteDataSource(this._apiClient);

  final BillingApiClient _apiClient;

  Future<DonationIntentDto> createDonationIntent({
    required String targetType,
    required String targetId,
    required int units,
    required int nanos,
    required String currencyCode,
    String provider = 'mock',
  }) {
    return _apiClient.createDonationIntent(
      targetType: targetType,
      targetId: targetId,
      units: units,
      nanos: nanos,
      currencyCode: currencyCode,
      provider: provider,
    );
  }
}
